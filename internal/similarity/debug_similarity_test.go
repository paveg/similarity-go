package similarity

import (
	"testing"
)

// TestSimilarityDebugging debugs why similarity calculations return 0.000.
func TestSimilarityDebugging(t *testing.T) {
	t.Log("🔍 Debugging Similarity Calculation Issues")

	// Create a simple detector
	detector := NewDetector(0.8)

	// Test with identical functions - should be 1.0
	dataset := GetBenchmarkDataset()
	identicalCase := dataset[0] // identical_functions case

	func1, func2 := identicalCase.CreateFunctionPair(t)

	// Check if functions are nil first
	if func1 == nil || func2 == nil {
		t.Fatalf("Functions are nil: func1=%v, func2=%v", func1, func2)
	}

	// Check if functions are created correctly
	t.Logf("Function 1: Name=%s, AST is nil=%t", func1.Name, func1.AST == nil)
	t.Logf("Function 2: Name=%s, AST is nil=%t", func2.Name, func2.AST == nil)

	if func1.AST != nil {
		source1, _ := func1.GetSource()
		t.Logf("Function 1 source length: %d characters", len(source1))
		t.Logf("Function 1 hash: %s", func1.Hash())
	}

	if func2.AST != nil {
		source2, _ := func2.GetSource()
		t.Logf("Function 2 source length: %d characters", len(source2))
		t.Logf("Function 2 hash: %s", func2.Hash())
	}

	// Test individual similarity components
	if func1.AST != nil && func2.AST != nil {
		// Test hash comparison (early termination check)
		hash1 := func1.Hash()
		hash2 := func2.Hash()
		t.Logf("Hash equality: %t (hash1=%s, hash2=%s)", hash1 == hash2, hash1, hash2)

		// Test couldBeSimilar check
		couldBe := detector.couldBeSimilar(func1, func2)
		t.Logf("Could be similar: %t", couldBe)

		// Test normalized AST comparison
		normalizedSame := detector.compareNormalizedAST(func1, func2)
		t.Logf("Normalized AST same: %t", normalizedSame)

		// Test individual algorithm components
		treeEditSim := detector.calculateTreeEditSimilarity(func1, func2)
		t.Logf("Tree edit similarity: %.4f", treeEditSim)

		tokenSim := TokenSequenceSimilarity(func1, func2)
		t.Logf("Token similarity: %.4f", tokenSim)

		structuralSim := detector.calculateStructuralSimilarity(func1, func2)
		t.Logf("Structural similarity: %.4f", structuralSim)

		signatureSim := detector.calculateSignatureSimilarity(func1, func2)
		t.Logf("Signature similarity: %.4f", signatureSim)

		// Calculate final weighted result manually
		weights := detector.config.Similarity.Weights
		result := weights.TreeEdit*treeEditSim + weights.TokenSimilarity*tokenSim +
			weights.Structural*structuralSim + weights.Signature*signatureSim
		t.Logf("Manual weighted result: %.4f", result)

		// Compare with actual CalculateSimilarity
		actualSim := detector.CalculateSimilarity(func1, func2)
		t.Logf("CalculateSimilarity result: %.4f", actualSim)
	}

	// Test with different functions to see if the pattern is consistent
	t.Log("\n--- Testing with different functions ---")
	differentCase := dataset[6] // completely_different case
	diffFunc1, diffFunc2 := differentCase.CreateFunctionPair(t)

	diffSim := detector.CalculateSimilarity(diffFunc1, diffFunc2)
	t.Logf("Different functions similarity: %.4f (expected: %.3f)", diffSim, differentCase.ExpectedSimilarity)

	// Test configuration
	t.Log("\n--- Configuration Check ---")
	t.Logf("Detector threshold: %.2f", detector.threshold)
	t.Logf("Tree edit weight: %.3f", detector.config.Similarity.Weights.TreeEdit)
	t.Logf("Token similarity weight: %.3f", detector.config.Similarity.Weights.TokenSimilarity)
	t.Logf("Structural weight: %.3f", detector.config.Similarity.Weights.Structural)
	t.Logf("Signature weight: %.3f", detector.config.Similarity.Weights.Signature)

	total := detector.config.Similarity.Weights.TreeEdit +
		detector.config.Similarity.Weights.TokenSimilarity +
		detector.config.Similarity.Weights.Structural +
		detector.config.Similarity.Weights.Signature
	t.Logf("Total weights: %.3f", total)
}
