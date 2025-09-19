package similarity

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/paveg/similarity-go/internal/config"
)

func TestConfigUpdater_NewConfigUpdater(t *testing.T) {
	updater := NewConfigUpdater()

	if updater == nil {
		t.Fatal("Expected non-nil updater")
	}

	if updater.backupDir == "" {
		t.Error("Expected non-empty backup directory")
	}
}

func TestConfigUpdater_ValidateWeightSum(t *testing.T) {
	updater := NewConfigUpdater()

	tests := []struct {
		name        string
		weights     config.SimilarityWeights
		expectError bool
	}{
		{
			name: "valid_weights",
			weights: config.SimilarityWeights{
				TreeEdit:        0.3,
				TokenSimilarity: 0.3,
				Structural:      0.25,
				Signature:       0.15,
			},
			expectError: false,
		},
		{
			name: "sum_too_low",
			weights: config.SimilarityWeights{
				TreeEdit:        0.2,
				TokenSimilarity: 0.2,
				Structural:      0.2,
				Signature:       0.2,
			}, // Sum = 0.8
			expectError: true,
		},
		{
			name: "sum_too_high",
			weights: config.SimilarityWeights{
				TreeEdit:        0.4,
				TokenSimilarity: 0.4,
				Structural:      0.3,
				Signature:       0.3,
			}, // Sum = 1.4
			expectError: true,
		},
		{
			name: "negative_weight",
			weights: config.SimilarityWeights{
				TreeEdit:        -0.1,
				TokenSimilarity: 0.4,
				Structural:      0.4,
				Signature:       0.3,
			},
			expectError: true,
		},
		{
			name: "zero_weight",
			weights: config.SimilarityWeights{
				TreeEdit:        0,
				TokenSimilarity: 0.4,
				Structural:      0.3,
				Signature:       0.3,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := updater.ValidateWeightSum(tt.weights)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateWeightSum() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestConfigUpdater_updateConstantLine(t *testing.T) {
	updater := NewConfigUpdater()

	tests := []struct {
		name         string
		line         string
		constantName string
		value        float64
		expected     string
	}{
		{
			name:         "simple_constant",
			line:         "\tTreeEditWeight = 0.3000",
			constantName: "TreeEditWeight",
			value:        0.4567,
			expected:     "\tTreeEditWeight = 0.4567 // Optimized " + time.Now().Format("2006-01-02"),
		},
		{
			name:         "no_indentation",
			line:         "TokenSimilarityWeight = 0.2500",
			constantName: "TokenSimilarityWeight",
			value:        0.1234,
			expected:     "TokenSimilarityWeight = 0.1234 // Optimized " + time.Now().Format("2006-01-02"),
		},
		{
			name:         "spaces_indentation",
			line:         "    StructuralWeight = 0.2000",
			constantName: "StructuralWeight",
			value:        0.3000,
			expected:     "    StructuralWeight = 0.3000 // Optimized " + time.Now().Format("2006-01-02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updater.updateConstantLine(tt.line, tt.constantName, tt.value)
			if result != tt.expected {
				t.Errorf("updateConstantLine() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestConfigUpdater_updateWeightConstants(t *testing.T) {
	updater := NewConfigUpdater()

	content := `package config

const (
	TreeEditWeight = 0.3000
	TokenSimilarityWeight = 0.2500
	StructuralWeight = 0.2000
	SignatureWeight = 0.2500
	DifferentSignatureWeight = 0.3000
)
`

	weights := config.SimilarityWeights{
		TreeEdit:           0.35,
		TokenSimilarity:    0.30,
		Structural:         0.25,
		Signature:          0.10,
		DifferentSignature: 0.30,
	}

	result := updater.updateWeightConstants(content, weights)

	// Check that all weights were updated
	if !strings.Contains(result, "TreeEditWeight = 0.3500") {
		t.Error("TreeEditWeight not updated correctly")
	}
	if !strings.Contains(result, "TokenSimilarityWeight = 0.3000") {
		t.Error("TokenSimilarityWeight not updated correctly")
	}
	if !strings.Contains(result, "StructuralWeight = 0.2500") {
		t.Error("StructuralWeight not updated correctly")
	}
	if !strings.Contains(result, "SignatureWeight = 0.1000") {
		t.Error("SignatureWeight not updated correctly")
	}
	if !strings.Contains(result, "DifferentSignatureWeight = 0.3000") {
		t.Error("DifferentSignatureWeight not updated correctly")
	}

	// Check optimization comment is added
	if !strings.Contains(result, "Weight optimization performed") {
		t.Error("Optimization comment not added")
	}

	// Check individual line comments are added
	todayStr := time.Now().Format("2006-01-02")
	if !strings.Contains(result, "// Optimized "+todayStr) {
		t.Error("Individual optimization comments not added")
	}
}

func TestConfigUpdater_CreateYAMLConfig(t *testing.T) {
	updater := NewConfigUpdater()

	// Create temporary directory for test
	tempDir := t.TempDir()
	yamlFile := filepath.Join(tempDir, "test_config.yaml")

	weights := config.SimilarityWeights{
		TreeEdit:           0.35,
		TokenSimilarity:    0.30,
		Structural:         0.25,
		Signature:          0.10,
		DifferentSignature: 0.30,
	}

	err := updater.CreateYAMLConfig(weights, yamlFile)
	if err != nil {
		t.Fatalf("CreateYAMLConfig() error = %v", err)
	}

	// Check file was created
	if _, statErr := os.Stat(yamlFile); os.IsNotExist(statErr) {
		t.Fatal("YAML config file was not created")
	}

	// Read and validate content
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	contentStr := string(content)

	// Check weights are present
	if !strings.Contains(contentStr, "tree_edit: 0.3500") {
		t.Error("TreeEdit weight not found in YAML")
	}
	if !strings.Contains(contentStr, "token_similarity: 0.3000") {
		t.Error("TokenSimilarity weight not found in YAML")
	}
	if !strings.Contains(contentStr, "structural: 0.2500") {
		t.Error("Structural weight not found in YAML")
	}
	if !strings.Contains(contentStr, "signature: 0.1000") {
		t.Error("Signature weight not found in YAML")
	}
	if !strings.Contains(contentStr, "different_signature: 0.3000") {
		t.Error("DifferentSignature weight not found in YAML")
	}

	// Check structure
	if !strings.Contains(contentStr, "similarity:") {
		t.Error("Similarity section not found")
	}
	if !strings.Contains(contentStr, "weights:") {
		t.Error("Weights section not found")
	}
	if !strings.Contains(contentStr, "thresholds:") {
		t.Error("Thresholds section not found")
	}
}

func TestConfigUpdater_generateUpdateSummary(t *testing.T) {
	updater := NewConfigUpdater()

	result := &UpdateResult{
		OldWeights: config.SimilarityWeights{
			TreeEdit:           0.30,
			TokenSimilarity:    0.25,
			Structural:         0.20,
			Signature:          0.25,
			DifferentSignature: 0.30,
		},
		NewWeights: config.SimilarityWeights{
			TreeEdit:           0.35,
			TokenSimilarity:    0.30,
			Structural:         0.25,
			Signature:          0.10,
			DifferentSignature: 0.30,
		},
		UpdatedFiles: []string{"config.go"},
		BackupFiles:  []string{"backup_config.go"},
		Timestamp:    time.Now(),
	}

	validation := ValidationResult{
		MAE:             0.05,
		RMSE:            0.08,
		PearsonR:        0.95,
		Accuracy:        0.90,
		F1Score:         0.88,
		RobustnessScore: 0.85,
	}

	summary := updater.generateUpdateSummary(result, validation)

	// Check required sections
	if !strings.Contains(summary, "CONFIGURATION UPDATE SUMMARY") {
		t.Error("Summary header not found")
	}
	if !strings.Contains(summary, "WEIGHT CHANGES") {
		t.Error("Weight changes section not found")
	}
	if !strings.Contains(summary, "PERFORMANCE IMPROVEMENT") {
		t.Error("Performance improvement section not found")
	}
	if !strings.Contains(summary, "FILES MODIFIED") {
		t.Error("Files modified section not found")
	}
	if !strings.Contains(summary, "BACKUP FILES") {
		t.Error("Backup files section not found")
	}
	if !strings.Contains(summary, "NEXT STEPS") {
		t.Error("Next steps section not found")
	}

	// Check weight changes
	if !strings.Contains(summary, "0.3000 → 0.3500 (+0.0500)") {
		t.Error("TreeEdit weight change not formatted correctly")
	}
	if !strings.Contains(summary, "0.2500 → 0.1000 (-0.1500)") {
		t.Error("Signature weight change not formatted correctly")
	}

	// Check performance metrics
	if !strings.Contains(summary, "0.050000") { // MAE
		t.Error("MAE not found in summary")
	}
	if !strings.Contains(summary, "0.950000") { // Pearson correlation
		t.Error("Pearson correlation not found in summary")
	}

	// Check file information
	if !strings.Contains(summary, "config.go") {
		t.Error("Updated file not listed")
	}
	if !strings.Contains(summary, "backup_config.go") {
		t.Error("Backup file not listed")
	}
}

func TestConfigUpdater_ensureBackupDir(t *testing.T) {
	// Use temporary directory for testing
	tempDir := t.TempDir()

	updater := &ConfigUpdater{
		backupDir: filepath.Join(tempDir, "test_backups"),
	}

	err := updater.ensureBackupDir()
	if err != nil {
		t.Fatalf("ensureBackupDir() error = %v", err)
	}

	// Check directory was created
	if _, statErr := os.Stat(updater.backupDir); os.IsNotExist(statErr) {
		t.Error("Backup directory was not created")
	}

	// Should not error if called again
	err = updater.ensureBackupDir()
	if err != nil {
		t.Errorf("ensureBackupDir() should not error on existing directory: %v", err)
	}
}

func TestConfigUpdater_ListBackups(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")

	updater := &ConfigUpdater{
		backupDir: backupDir,
	}

	// Create backup directory and some test backup files
	err := os.MkdirAll(backupDir, 0o750)
	if err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}

	// Create test backup files
	backupFiles := []string{
		"config_20230101_120000.go",
		"config_20230102_130000.go",
		"config_20230103_140000.go",
	}

	for _, filename := range backupFiles {
		writeErr := os.WriteFile(filepath.Join(backupDir, filename), []byte("test content"), 0o600)
		if writeErr != nil {
			t.Fatalf("Failed to create test backup file: %v", writeErr)
		}
	}

	// Create a non-matching file to ensure it's not included
	err = os.WriteFile(filepath.Join(backupDir, "other_file.txt"), []byte("other"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	// Test ListBackups
	backups, err := updater.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error = %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backup files, got %d", len(backups))
	}

	// Check each backup file is present
	for _, expectedFile := range backupFiles {
		found := false
		expectedPath := filepath.Join(backupDir, expectedFile)
		for _, backup := range backups {
			if backup == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected backup file %s not found in results", expectedFile)
		}
	}
}

func TestConfigUpdater_addOptimizationComment(t *testing.T) {
	updater := NewConfigUpdater()

	content := `package config

import "fmt"

const TreeEditWeight = 0.3`

	weights := config.SimilarityWeights{
		TreeEdit:        0.35,
		TokenSimilarity: 0.30,
		Structural:      0.25,
		Signature:       0.10,
	}

	result := updater.addOptimizationComment(content, weights)

	// Check that comment was added after package declaration
	lines := strings.Split(result, "\n")

	// Find package line
	packageIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			packageIndex = i
			break
		}
	}

	if packageIndex == -1 {
		t.Fatal("Package declaration not found")
	}

	// Check that optimization comment appears after package line
	commentFound := false
	for i := packageIndex + 1; i < len(lines); i++ {
		if strings.Contains(lines[i], "Weight optimization performed") {
			commentFound = true
			break
		}
	}

	if !commentFound {
		t.Error("Optimization comment not found after package declaration")
	}

	// Check comment contains expected information
	if !strings.Contains(result, "Weight optimization performed") {
		t.Error("Optimization comment header not found")
	}
	if !strings.Contains(result, "TreeEdit: 0.3500") {
		t.Error("TreeEdit weight not found in comment")
	}
	if !strings.Contains(result, "TokenSimilarity: 0.3000") {
		t.Error("TokenSimilarity weight not found in comment")
	}
}

func TestConfigUpdater_ValidationIntegration(t *testing.T) {
	// Test integration between validation and configuration update
	updater := NewConfigUpdater()

	weights := config.SimilarityWeights{
		TreeEdit:           0.35,
		TokenSimilarity:    0.30,
		Structural:         0.25,
		Signature:          0.10,
		DifferentSignature: 0.30,
	}

	// Validate weights before attempting update
	err := updater.ValidateWeightSum(weights)
	if err != nil {
		t.Fatalf("Weight validation failed: %v", err)
	}

	// Create mock validation result
	validationResult := ValidationResult{
		MAE:             0.045,
		RMSE:            0.067,
		PearsonR:        0.92,
		Accuracy:        0.88,
		F1Score:         0.85,
		RobustnessScore: 0.83,
	}

	// Test YAML config generation
	tempDir := t.TempDir()
	yamlFile := filepath.Join(tempDir, "optimized_config.yaml")

	err = updater.CreateYAMLConfig(weights, yamlFile)
	if err != nil {
		t.Fatalf("YAML config creation failed: %v", err)
	}

	// Verify YAML file exists and has content
	content, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}

	if len(content) == 0 {
		t.Error("YAML file is empty")
	}

	// Test summary generation
	result := &UpdateResult{
		OldWeights: config.SimilarityWeights{
			TreeEdit:           0.30,
			TokenSimilarity:    0.25,
			Structural:         0.20,
			Signature:          0.25,
			DifferentSignature: 0.30,
		},
		NewWeights:   weights,
		UpdatedFiles: []string{yamlFile},
		BackupFiles:  []string{},
		Timestamp:    time.Now(),
	}

	summary := updater.generateUpdateSummary(result, validationResult)

	if len(summary) == 0 {
		t.Error("Update summary is empty")
	}

	// Summary should contain key performance metrics
	expectedMetrics := []string{"0.045000", "0.067000", "0.920000", "0.880000"}
	for _, metric := range expectedMetrics {
		if !strings.Contains(summary, metric) {
			t.Errorf("Summary missing expected metric: %s", metric)
		}
	}
}
