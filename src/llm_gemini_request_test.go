package main

import "testing"

func TestBuildGeminiRequestRequiresConversationContent(t *testing.T) {
	req, roles, err := buildGeminiRequest([]Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "assistant", Content: ""},
	}, nil)
	if err == nil {
		t.Fatalf("expected error for zero conversation contents, got request=%+v roles=%v", req, roles)
	}
}

func TestBuildGeminiRequestTrimsLeadingModelMessages(t *testing.T) {
	req, _, err := buildGeminiRequest([]Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "assistant", Content: "stale assistant turn"},
		{Role: "user", Content: "hello"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 content after trimming, got %d", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Fatalf("first content role = %q, want user", req.Contents[0].Role)
	}
	if len(req.Contents[0].Parts) != 1 || req.Contents[0].Parts[0].Text != "hello" {
		t.Fatalf("unexpected user content payload: %+v", req.Contents[0])
	}
}

func TestBuildGeminiRequestIncludesWorkflowSingleShotPrompt(t *testing.T) {
	req, _, err := buildGeminiRequest([]Message{
		{Role: "system", Content: "environment context"},
		{Role: "system", Content: "node prompt"},
		{Role: "user", Content: "prompt:\nHey."},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Contents) != 1 {
		t.Fatalf("expected 1 user content, got %d", len(req.Contents))
	}
	if req.Contents[0].Role != "user" {
		t.Fatalf("content role = %q, want user", req.Contents[0].Role)
	}
	if len(req.Contents[0].Parts) != 1 || req.Contents[0].Parts[0].Text == "" {
		t.Fatalf("expected non-empty text part, got %+v", req.Contents[0])
	}
	if req.SystemInstruction == nil || len(req.SystemInstruction.Parts) != 2 {
		t.Fatalf("expected 2 system instruction parts, got %+v", req.SystemInstruction)
	}
}
