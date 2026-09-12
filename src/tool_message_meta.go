package main

func buildToolMessageMeta(toolName, result string) map[string]interface{} {
	artifacts := inferArtifactsFromToolResult(toolName, result)
	if len(artifacts) == 0 {
		return nil
	}
	return map[string]interface{}{
		"artifacts": artifacts,
	}
}
