package usecase

import (
	"strings"
	"testing"
)

func TestPersonaGenerator_Generate(t *testing.T) {
	pg := NewPersonaGenerator()

	// Test 1: Basic Generation
	user1 := pg.Generate()
	if user1.PersonaName == "" {
		t.Error("Expected persona name to be non-empty")
	}
	if user1.PersonaColor == "" {
		t.Error("Expected persona color to be non-empty")
	}

	// Test 2: Uniqueness
	user2 := pg.Generate()
	if user1.PersonaName == user2.PersonaName {
		t.Error("Expected unique persona names", user1.PersonaName, user2.PersonaName)
	}
}

func TestPersonaGenerator_Format(t *testing.T) {
	pg := NewPersonaGenerator()
	user := pg.Generate()
	
	parts := strings.Split(user.PersonaName, " ")
	if len(parts) < 2 {
		t.Errorf("Expected name format 'Noun Adjective', got: %s", user.PersonaName)
	}
}

func TestPersonaGenerator_Release(t *testing.T) {
	pg := NewPersonaGenerator()
	
	// Fill up one name (mocking strictly isn't easy without exposing dependency, 
	// but we can test logic by releasing and checking usage count indirectly if we had a getter, 
	// or just ensuring Release doesn't panic)
	
	user := pg.Generate()
	name := user.PersonaName
	
	if !pg.existing[name] {
		t.Error("Expected name to be marked as existing")
	}

	pg.Release(name)
	
	if pg.existing[name] {
		t.Error("Expected name to be released (removed from map)")
	}
}

func TestPersonaGenerator_Concurrency(t *testing.T) {
	pg := NewPersonaGenerator()
	
	// Generate 1000 names concurrently looking for race conditions
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			pg.Generate()
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
	
	if len(pg.existing) != 100 {
		t.Errorf("Expected 100 existing names, got %d", len(pg.existing))
	}
}
