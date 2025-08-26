package samples

// This file has intentional syntax errors for testing error handling

func InvalidFunction( {
	return "missing parameter"
}

func AnotherFunction() string
	return "missing opening brace"
}

func IncompleteFunction() {
	// Missing closing brace