package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// MCP JSON-RPC 2.0 Structures
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type MCPToolListResult struct {
	Tools []MCPTool `json:"tools"`
}

type MCPTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type MCPToolCallParams struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments"`
}

type MCPToolCallResult struct {
	Content []MCPTextContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type MCPTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// RunningMCPServer tracks background child servers
type RunningMCPServer struct {
	Name   string
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
}

var activeMCPServers []RunningMCPServer

// bootstrapMCPServers spawns each local MCP server and performs the handshake
func bootstrapMCPServers() {
	if len(activeConfig.MCPServers) == 0 {
		return
	}

	fmt.Printf("%s[MCP] Bootstrapping registered Model Context Protocol servers...%s\n", ColorBold+ColorBlue, ColorReset)

	for name, srv := range activeConfig.MCPServers {
		fmt.Printf("[MCP] Spawning '%s' using: %s %s...\n", name, srv.Command, strings.Join(srv.Args, " "))
		
		cmd := exec.Command(srv.Command, srv.Args...)
		cmd.Env = os.Environ()
		for k, v := range srv.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			fmt.Printf("%s[MCP ERROR] Failed to create stdin pipe for '%s': %v%s\n", ColorRed, name, err, ColorReset)
			continue
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Printf("%s[MCP ERROR] Failed to create stdout pipe for '%s': %v%s\n", ColorRed, name, err, ColorReset)
			continue
		}

		err = cmd.Start()
		if err != nil {
			fmt.Printf("%s[MCP WARNING] Could not start server '%s': %v (Is command available on PATH?)%s\n", ColorYellow, name, err, ColorReset)
			continue
		}

		runningServer := RunningMCPServer{
			Name:   name,
			Cmd:    cmd,
			Stdin:  stdinPipe,
			Stdout: stdoutPipe,
		}

		err = performMCPHandshake(&runningServer)
		if err != nil {
			fmt.Printf("%s[MCP ERROR] Handshake failed with '%s': %v. Terminating server.%s\n", ColorRed, name, err, ColorReset)
			_ = cmd.Process.Kill()
			continue
		}

		activeMCPServers = append(activeMCPServers, runningServer)
		fmt.Printf("%s[MCP SUCCESS] Server '%s' initialized and ready.%s\n", ColorGreen, name, ColorReset)
	}
}

// performMCPHandshake conducts the protocol compliance handshakes (Initialize ➔ Initialized)
func performMCPHandshake(srv *RunningMCPServer) error {
	initParams := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    make(map[string]interface{}),
		ClientInfo: ClientInfo{
			Name:    "goharness-client",
			Version: "1.0.0",
		},
	}

	resp, err := callMCPServer(srv.Stdin, srv.Stdout, "initialize", initParams, 1)
	if err != nil {
		return fmt.Errorf("initialize call failed: %w", err)
	}

	if resp.Error != nil {
		return fmt.Errorf("server returned initialize error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	err = notifyMCPServer(srv.Stdin, "notifications/initialized", nil)
	if err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	return nil
}

// callMCPServer writes a single-line JSON-RPC request to stdin, and blocks to read a single-line reply from stdout
func callMCPServer(stdin io.WriteCloser, stdout io.Reader, method string, params interface{}, id int) (*JSONRPCResponse, error) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	_, err = stdin.Write(append(reqBytes, '\n'))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		line := scanner.Bytes()
		var resp JSONRPCResponse
		err = json.Unmarshal(line, &resp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JSON-RPC response: %w. Raw: %s", err, string(line))
		}
		return &resp, nil
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning server stdout: %w", err)
	}

	return nil, fmt.Errorf("received empty EOF from server standard output")
}

// notifyMCPServer writes a standard JSON-RPC notification (no ID) which expects no response
func notifyMCPServer(stdin io.WriteCloser, method string, params interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = stdin.Write(append(reqBytes, '\n'))
	return err
}

// discoverMCPTools queries each active server for its tools and updates the active list
func discoverMCPTools() []Tool {
	var mcpTools []Tool

	for _, srv := range activeMCPServers {
		resp, err := callMCPServer(srv.Stdin, srv.Stdout, "tools/list", nil, 2)
		if err != nil {
			fmt.Printf("%s[MCP WARNING] Failed to retrieve tools list from '%s': %v%s\n", ColorYellow, srv.Name, err, ColorReset)
			continue
		}

		if resp.Error != nil {
			fmt.Printf("%s[MCP WARNING] Server '%s' returned tools/list error: %s%s\n", ColorYellow, srv.Name, resp.Error.Message, ColorReset)
			continue
		}

		var listResult MCPToolListResult
		err = json.Unmarshal(resp.Result, &listResult)
		if err != nil {
			fmt.Printf("%s[MCP WARNING] Failed to unmarshal tools/list from '%s': %v%s\n", ColorYellow, srv.Name, err, ColorReset)
			continue
		}

		for _, tool := range listResult.Tools {
			mcpToolsMap[tool.Name] = srv.Name

			apiTool := Tool{
				Type: "function",
				Function: FunctionDescriptor{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			}
			mcpTools = append(mcpTools, apiTool)
			fmt.Printf("  ➔ Discovered MCP Tool from '%s': %s (%s)\n", srv.Name, tool.Name, strings.Split(tool.Description, "\n")[0])
		}
	}

	return mcpTools
}

// executeMCPToolCall forwards the LLM tool execution command directly to the target child process
func executeMCPToolCall(serverName, toolName, argumentsJson string) string {
	var srv *RunningMCPServer
	for i := range activeMCPServers {
		if activeMCPServers[i].Name == serverName {
			srv = &activeMCPServers[i]
			break
		}
	}

	if srv == nil {
		return fmt.Sprintf("Error: Target MCP server '%s' is not running.", serverName)
	}

	var parsedArgs interface{}
	err := json.Unmarshal([]byte(argumentsJson), &parsedArgs)
	if err != nil {
		return fmt.Sprintf("Error: Failed to parse tool arguments JSON: %v", err)
	}

	callParams := MCPToolCallParams{
		Name:      toolName,
		Arguments: parsedArgs,
	}

	resp, err := callMCPServer(srv.Stdin, srv.Stdout, "tools/call", callParams, 3)
	if err != nil {
		return fmt.Sprintf("MCP execution failure: %v", err)
	}

	if resp.Error != nil {
		return fmt.Sprintf("MCP Server Error (code %d): %s", resp.Error.Code, resp.Error.Message)
	}

	var callResult MCPToolCallResult
	err = json.Unmarshal(resp.Result, &callResult)
	if err != nil {
		return fmt.Sprintf("MCP Decoding failure: %v. Raw Result: %s", err, string(resp.Result))
	}

	var out strings.Builder
	for _, content := range callResult.Content {
		if content.Type == "text" {
			out.WriteString(content.Text)
			out.WriteString("\n")
		}
	}

	if out.Len() == 0 {
		return "(Tool executed successfully with empty stdout outcome)"
	}

	return out.String()
}

func cleanupMCPServers() {
	if len(activeMCPServers) == 0 {
		return
	}
	fmt.Printf("\n%s[MCP] Cleaning up and terminating child server processes...%s\n", ColorBold+ColorBlue, ColorReset)
	for _, srv := range activeMCPServers {
		if srv.Cmd != nil && srv.Cmd.Process != nil {
			fmt.Printf("  ↳ Killing '%s' (PID %d)\n", srv.Name, srv.Cmd.Process.Pid)
			_ = srv.Cmd.Process.Kill()
		}
	}
}
