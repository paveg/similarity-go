package similarity

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/paveg/similarity-go/internal/config"
)

// ConfigUpdater handles updating configuration files with optimized weights.
type ConfigUpdater struct {
	backupDir string
}

// NewConfigUpdater creates a new configuration updater.
func NewConfigUpdater() *ConfigUpdater {
	return &ConfigUpdater{
		backupDir: ".similarity_backups",
	}
}

// UpdateResult contains the results of a configuration update.
type UpdateResult struct {
	UpdatedFiles  []string
	BackupFiles   []string
	OldWeights    config.SimilarityWeights
	NewWeights    config.SimilarityWeights
	Timestamp     time.Time
	UpdateSummary string
}

// UpdateConfigWithWeights updates configuration files with new optimized weights.
func (cu *ConfigUpdater) UpdateConfigWithWeights(
	newWeights config.SimilarityWeights,
	validationResult ValidationResult,
) (*UpdateResult, error) {
	result := &UpdateResult{
		OldWeights: config.SimilarityWeights{
			TreeEdit:           config.TreeEditWeight,
			TokenSimilarity:    config.TokenSimilarityWeight,
			Structural:         config.StructuralWeight,
			Signature:          config.SignatureWeight,
			DifferentSignature: config.DifferentSignatureWeight,
		},
		NewWeights: newWeights,
		Timestamp:  time.Now(),
	}

	// Create backup directory
	if err := cu.ensureBackupDir(); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Update config.go file with new constants
	if err := cu.updateConfigFile(newWeights, result); err != nil {
		return nil, fmt.Errorf("failed to update config file: %w", err)
	}

	// Generate update summary
	result.UpdateSummary = cu.generateUpdateSummary(result, validationResult)

	return result, nil
}

// ensureBackupDir creates the backup directory if it doesn't exist.
func (cu *ConfigUpdater) ensureBackupDir() error {
	return os.MkdirAll(cu.backupDir, 0o750)
}

// updateConfigFile updates the main config.go file with new weight constants.
func (cu *ConfigUpdater) updateConfigFile(weights config.SimilarityWeights, result *UpdateResult) error {
	configFile := "internal/config/config.go"

	// Read current config file
	content, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Create backup
	backupFile := filepath.Join(cu.backupDir, fmt.Sprintf("config_%s.go",
		result.Timestamp.Format("20060102_150405")))

	if writeErr := os.WriteFile(backupFile, content, 0o600); writeErr != nil {
		return fmt.Errorf("failed to create backup: %w", writeErr)
	}
	result.BackupFiles = append(result.BackupFiles, backupFile)

	// Update weight constants
	updatedContent := cu.updateWeightConstants(string(content), weights)

	// Write updated config
	if writeErr := os.WriteFile(configFile, []byte(updatedContent), 0o600); writeErr != nil {
		return fmt.Errorf("failed to write updated config: %w", writeErr)
	}
	result.UpdatedFiles = append(result.UpdatedFiles, configFile)

	return nil
}

// updateWeightConstants replaces weight constants in the config file content.
func (cu *ConfigUpdater) updateWeightConstants(content string, weights config.SimilarityWeights) string {
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		containsAssignment := strings.Contains(trimmed, "=")

		switch {
		case strings.Contains(trimmed, "TreeEditWeight") && containsAssignment:
			lines[i] = cu.updateConstantLine(line, "TreeEditWeight", weights.TreeEdit)
		case strings.Contains(trimmed, "TokenSimilarityWeight") && containsAssignment:
			lines[i] = cu.updateConstantLine(line, "TokenSimilarityWeight", weights.TokenSimilarity)
		case strings.Contains(trimmed, "StructuralWeight") && containsAssignment:
			lines[i] = cu.updateConstantLine(line, "StructuralWeight", weights.Structural)
		case strings.Contains(trimmed, "SignatureWeight") && containsAssignment && !strings.Contains(trimmed, "Different"):
			lines[i] = cu.updateConstantLine(line, "SignatureWeight", weights.Signature)
		case strings.Contains(trimmed, "DifferentSignatureWeight") && containsAssignment:
			lines[i] = cu.updateConstantLine(line, "DifferentSignatureWeight", weights.DifferentSignature)
		}
	}

	// Add optimization comment
	updatedContent := strings.Join(lines, "\n")
	return cu.addOptimizationComment(updatedContent, weights)
}

// updateConstantLine updates a single constant declaration line.
func (cu *ConfigUpdater) updateConstantLine(line, constantName string, value float64) string {
	// Find the indentation
	indent := ""
	for _, char := range line {
		if char == ' ' || char == '\t' {
			indent += string(char)
		} else {
			break
		}
	}

	// Generate new line with comment
	timestamp := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s%s = %.4f // Optimized %s", indent, constantName, value, timestamp)
}

// addOptimizationComment adds a comment block about the optimization.
func (cu *ConfigUpdater) addOptimizationComment(content string, weights config.SimilarityWeights) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	comment := fmt.Sprintf(`
// Weight optimization performed on %s
// Optimized weights sum: %.4f
// TreeEdit: %.4f, TokenSimilarity: %.4f, Structural: %.4f, Signature: %.4f
`, timestamp,
		weights.TreeEdit+weights.TokenSimilarity+weights.Structural+weights.Signature,
		weights.TreeEdit, weights.TokenSimilarity, weights.Structural, weights.Signature)

	// Find a good place to insert the comment (after package declaration)
	lines := strings.Split(content, "\n")
	insertIndex := 0

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "package ") {
			insertIndex = i + 1
			break
		}
	}

	// Insert comment after package declaration
	result := make([]string, 0, len(lines)+strings.Count(comment, "\n"))
	result = append(result, lines[:insertIndex]...)
	result = append(result, strings.Split(comment, "\n")...)
	result = append(result, lines[insertIndex:]...)

	return strings.Join(result, "\n")
}

// generateUpdateSummary creates a human-readable summary of the update.
func (cu *ConfigUpdater) generateUpdateSummary(result *UpdateResult, validation ValidationResult) string {
	var summary strings.Builder

	summary.WriteString("=== CONFIGURATION UPDATE SUMMARY ===\n")
	summary.WriteString(fmt.Sprintf("Update Time: %s\n", result.Timestamp.Format("2006-01-02 15:04:05")))
	summary.WriteString(fmt.Sprintf("Files Updated: %d\n", len(result.UpdatedFiles)))
	summary.WriteString(fmt.Sprintf("Backup Files Created: %d\n", len(result.BackupFiles)))

	summary.WriteString("\n--- WEIGHT CHANGES ---\n")
	summary.WriteString(fmt.Sprintf("TreeEdit:        %.4f → %.4f (%+.4f)\n",
		result.OldWeights.TreeEdit, result.NewWeights.TreeEdit,
		result.NewWeights.TreeEdit-result.OldWeights.TreeEdit))
	summary.WriteString(fmt.Sprintf("TokenSimilarity: %.4f → %.4f (%+.4f)\n",
		result.OldWeights.TokenSimilarity, result.NewWeights.TokenSimilarity,
		result.NewWeights.TokenSimilarity-result.OldWeights.TokenSimilarity))
	summary.WriteString(fmt.Sprintf("Structural:      %.4f → %.4f (%+.4f)\n",
		result.OldWeights.Structural, result.NewWeights.Structural,
		result.NewWeights.Structural-result.OldWeights.Structural))
	summary.WriteString(fmt.Sprintf("Signature:       %.4f → %.4f (%+.4f)\n",
		result.OldWeights.Signature, result.NewWeights.Signature,
		result.NewWeights.Signature-result.OldWeights.Signature))

	summary.WriteString("\n--- PERFORMANCE IMPROVEMENT ---\n")
	summary.WriteString(fmt.Sprintf("Mean Absolute Error: %.6f\n", validation.MAE))
	summary.WriteString(fmt.Sprintf("Root Mean Square Error: %.6f\n", validation.RMSE))
	summary.WriteString(fmt.Sprintf("Pearson Correlation: %.6f\n", validation.PearsonR))
	summary.WriteString(fmt.Sprintf("Classification Accuracy: %.6f\n", validation.Accuracy))
	summary.WriteString(fmt.Sprintf("F1 Score: %.6f\n", validation.F1Score))
	summary.WriteString(fmt.Sprintf("Robustness Score: %.6f\n", validation.RobustnessScore))

	summary.WriteString("\n--- FILES MODIFIED ---\n")
	for _, file := range result.UpdatedFiles {
		summary.WriteString(fmt.Sprintf("Updated: %s\n", file))
	}

	summary.WriteString("\n--- BACKUP FILES ---\n")
	for _, file := range result.BackupFiles {
		summary.WriteString(fmt.Sprintf("Backup: %s\n", file))
	}

	summary.WriteString("\n--- NEXT STEPS ---\n")
	summary.WriteString("1. Test the updated configuration thoroughly\n")
	summary.WriteString("2. Run the full test suite to ensure compatibility\n")
	summary.WriteString("3. Benchmark performance on real codebases\n")
	summary.WriteString("4. Consider reverting if performance degrades\n")

	return summary.String()
}

// CreateYAMLConfig creates a YAML configuration file with the new weights.
func (cu *ConfigUpdater) CreateYAMLConfig(weights config.SimilarityWeights, filename string) error {
	yamlContent := fmt.Sprintf(`# Similarity Detection Configuration
# Generated on %s

similarity:
  weights:
    tree_edit: %.4f        # Weight for AST tree edit distance
    token_similarity: %.4f # Weight for token sequence similarity  
    structural: %.4f       # Weight for structural similarity
    signature: %.4f        # Weight for function signature similarity
    different_signature: %.4f # Penalty for different signatures

  thresholds:
    high_similarity: 0.8   # Threshold for high similarity detection
    medium_similarity: 0.6 # Threshold for medium similarity detection
    low_similarity: 0.4    # Threshold for low similarity detection

# Algorithm parameters
detection:
  min_function_lines: 3    # Minimum lines for function to be analyzed
  max_comparison_depth: 10 # Maximum AST comparison depth
  enable_parallel: true    # Enable parallel processing
  
# Output configuration
output:
  format: json            # Output format: json, yaml, text
  include_details: true   # Include detailed similarity breakdown
  sort_by: similarity     # Sort results by: similarity, file, function
  
# Performance tuning
performance:
  max_workers: 4          # Maximum parallel workers
  chunk_size: 100         # Files per processing chunk
  cache_enabled: true     # Enable similarity calculation caching
`,
		time.Now().Format("2006-01-02 15:04:05"),
		weights.TreeEdit,
		weights.TokenSimilarity,
		weights.Structural,
		weights.Signature,
		weights.DifferentSignature)

	return os.WriteFile(filename, []byte(yamlContent), 0o600)
}

// RevertConfig reverts configuration changes using the most recent backup.
func (cu *ConfigUpdater) RevertConfig() error {
	// Find most recent backup
	backupFiles, err := filepath.Glob(filepath.Join(cu.backupDir, "config_*.go"))
	if err != nil {
		return fmt.Errorf("failed to find backup files: %w", err)
	}

	if len(backupFiles) == 0 {
		return errors.New("no backup files found")
	}

	// Sort to get most recent (assuming timestamp format)
	var mostRecent string
	for _, file := range backupFiles {
		if mostRecent == "" || file > mostRecent {
			mostRecent = file
		}
	}

	// Read backup content
	content, err := os.ReadFile(mostRecent)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Restore original config
	configFile := "internal/config/config.go"
	if writeErr := os.WriteFile(configFile, content, 0o600); writeErr != nil {
		return fmt.Errorf("failed to restore config file: %w", writeErr)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Configuration reverted from backup: %s\n", mostRecent)
	return nil
}

// ListBackups returns a list of available backup files.
func (cu *ConfigUpdater) ListBackups() ([]string, error) {
	backupFiles, err := filepath.Glob(filepath.Join(cu.backupDir, "config_*.go"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}

	return backupFiles, nil
}

// ValidateWeightSum ensures weights sum to approximately 1.0.
func (cu *ConfigUpdater) ValidateWeightSum(weights config.SimilarityWeights) error {
	sum := weights.TreeEdit + weights.TokenSimilarity + weights.Structural + weights.Signature

	if sum < 0.98 || sum > 1.02 {
		return fmt.Errorf("weights sum to %.6f, expected ~1.0", sum)
	}

	if weights.TreeEdit <= 0 || weights.TokenSimilarity <= 0 ||
		weights.Structural <= 0 || weights.Signature <= 0 {
		return errors.New("all weights must be positive")
	}

	return nil
}

// PrintUpdateReport prints a detailed report of the configuration update.
func (cu *ConfigUpdater) PrintUpdateReport(result *UpdateResult) {
	_, _ = fmt.Fprint(os.Stdout, result.UpdateSummary)
}
