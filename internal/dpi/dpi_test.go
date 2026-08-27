package dpi

import (
	"os"
	"path/filepath"
	"testing"

	"parentcontrol/internal/models"
)

func TestDPIAppCRUD(t *testing.T) {
	tmpDir := t.TempDir()
	featPath := filepath.Join(tmpDir, "feature.cfg")
	_ = os.WriteFile(featPath, []byte("#version 1.0\n#class chat 1 社交\n1001 QQ: [tcp]\n"), 0644)

	mgr := NewDPIManager(featPath)
	if len(mgr.GetCategories()) == 0 {
		t.Fatalf("expected categories to be loaded")
	}

	// 1. Add app
	newApp := models.AppInfo{
		Name:        "Test Game",
		ClassID:     1,
		Description: "A test game app",
	}
	added, err := mgr.AddApp(newApp)
	if err != nil {
		t.Fatalf("failed to add app: %v", err)
	}
	if added.ID <= 0 {
		t.Errorf("expected generated ID > 0, got %d", added.ID)
	}
	if !added.IsCustom {
		t.Errorf("expected IsCustom to be true")
	}

	// Verify get
	got, ok := mgr.GetApp(added.ID)
	if !ok || got.Name != "Test Game" {
		t.Errorf("expected to find app Test Game, got %+v", got)
	}

	// 2. Update app
	added.Name = "Updated Game"
	added.Description = "Updated description"
	updated, err := mgr.UpdateApp(added.ID, added)
	if err != nil {
		t.Fatalf("failed to update app: %v", err)
	}
	if updated.Name != "Updated Game" || updated.Description != "Updated description" {
		t.Errorf("expected updated fields, got %+v", updated)
	}

	// 3. Delete app
	if err := mgr.DeleteApp(added.ID); err != nil {
		t.Fatalf("failed to delete app: %v", err)
	}
	if _, ok := mgr.GetApp(added.ID); ok {
		t.Errorf("expected app to be deleted")
	}
}

func TestDPICategoryCRUD(t *testing.T) {
	mgr := NewDPIManager("/nonexistent/path/features.cfg")

	// 1. Add custom category
	cat := models.AppCategory{
		ClassZh: "AI 工具",
		Icon:    "bot",
	}
	addedCat, err := mgr.AddCategory(cat)
	if err != nil {
		t.Fatalf("failed to add category: %v", err)
	}
	if addedCat.ClassID <= 0 {
		t.Errorf("expected valid ClassID, got %d", addedCat.ClassID)
	}

	// 2. Add an app to this category
	app := models.AppInfo{
		Name:    "ChatGPT",
		ClassID: addedCat.ClassID,
	}
	addedApp, err := mgr.AddApp(app)
	if err != nil {
		t.Fatalf("failed to add app: %v", err)
	}
	if addedApp.ClassZh != "AI 工具" {
		t.Errorf("expected ClassZh to be 'AI 工具', got '%s'", addedApp.ClassZh)
	}

	// 3. Delete category
	if err := mgr.DeleteCategory(addedCat.ClassID); err != nil {
		t.Fatalf("failed to delete category: %v", err)
	}

	// App should now be reassigned to other (ClassID 99)
	appAfter, ok := mgr.GetApp(addedApp.ID)
	if !ok {
		t.Fatalf("expected app to still exist after category deletion")
	}
	if appAfter.ClassID != 99 {
		t.Errorf("expected app to be reclassified to 99, got %d", appAfter.ClassID)
	}
}
