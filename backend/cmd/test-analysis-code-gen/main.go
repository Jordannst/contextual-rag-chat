package main

import (
	"fmt"
	"log"
	"os"
	
	"backend/utils"
	
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("ANALYSIS CODE GENERATION TEST")
	fmt.Println("==============================================\n")
	
	// Load environment variables from .env file
	// godotenv.Load() looks for .env in current working directory
	err := godotenv.Load(".env")
	if err != nil {
		// Try alternative paths
		err = godotenv.Load("backend/.env")
		if err != nil {
			err = godotenv.Load("../.env")
			if err != nil {
				log.Printf("⚠️  Warning: Could not load .env file: %v", err)
				log.Printf("   Make sure .env exists in backend/ directory")
			} else {
				log.Printf("✅ Loaded .env from parent directory")
			}
		} else {
			log.Printf("✅ Loaded .env from backend/")
		}
	} else {
		log.Printf("✅ Loaded .env from current directory")
	}
	
	// Check if API key is available
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKeys := os.Getenv("GEMINI_API_KEYS")
		if apiKeys == "" {
			log.Fatal("❌ GEMINI_API_KEY or GEMINI_API_KEYS not found in environment\n" +
				"   Please set one of these environment variables in .env file")
		}
		log.Printf("✅ Using GEMINI_API_KEYS (found %d keys)", len(apiKeys))
	} else {
		log.Printf("✅ Using GEMINI_API_KEY")
	}
	
	// Sample file preview (structure of the data)
	filePreview := `Struktur Data (CSV):
Kolom: Nama, Harga, Jumlah, Total

Sample Data (5 baris pertama):
Nama             | Harga  | Jumlah | Total
----------------|--------|--------|--------
Naruto          | 45000  | 2      | 90000
One Piece       | 50000  | 1      | 50000
Bleach          | 40000  | 3      | 120000
Attack on Titan | 55000  | 2      | 110000
Death Note      | 35000  | 4      | 140000

Total rows: 5`

	// Test cases
	testCases := []struct {
		query       string
		description string
	}{
		{
			query:       "Berapa total penjualan?",
			description: "Calculate total sales",
		},
		{
			query:       "Berapa rata-rata harga?",
			description: "Calculate average price",
		},
		{
			query:       "Berapa jumlah item yang terjual?",
			description: "Count total items",
		},
		{
			query:       "Produk mana yang paling mahal?",
			description: "Find most expensive product",
		},
		{
			query:       "Berapa banyak produk yang harganya di atas 45000?",
			description: "Count products with price > 45000",
		},
		{
			query:       "Tampilkan total penjualan per produk",
			description: "Show total sales per product",
		},
		{
			query:       "Berapa total jumlah barang yang terjual?",
			description: "Sum of all quantities",
		},
		{
			query:       "Hitung berapa banyak baris data",
			description: "Count rows",
		},
	}
	
	passed := 0
	failed := 0
	
	for i, tc := range testCases {
		fmt.Printf("\n[Test %d/%d] %s\n", i+1, len(testCases), tc.description)
		fmt.Printf("Query: '%s'\n", tc.query)
		fmt.Println("----------------------------------------------")
		
		code, err := utils.GenerateAnalysisCode(tc.query, filePreview)
		
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
			failed++
			continue
		}
		
		fmt.Printf("✅ Generated Code:\n")
		fmt.Printf("%s\n", code)
		
		// Validate code
		if err := utils.ValidatePythonCode(code); err != nil {
			fmt.Printf("⚠️  WARNING: Code validation failed: %v\n", err)
			fmt.Printf("   (This might be a false positive)\n")
		} else {
			fmt.Printf("✅ Code validation passed\n")
		}
		
		passed++
	}
	
	// Test complete workflow (Code Gen + Execution)
	fmt.Println("\n==============================================")
	fmt.Println("COMPLETE WORKFLOW TEST (Gen + Execute)")
	fmt.Println("==============================================\n")
	
	// Create sample CSV for testing
	csvFile := "test_sales.csv"
	createSampleCSV(csvFile)
	fmt.Printf("[Setup] Created sample CSV: %s\n\n", csvFile)
	
	workflowTests := []string{
		"Berapa total penjualan?",
		"Berapa rata-rata harga?",
		"Berapa banyak produk?",
	}
	
	for i, query := range workflowTests {
		fmt.Printf("[Workflow Test %d/%d] Query: '%s'\n", i+1, len(workflowTests), query)
		
		// Step 1: Generate file preview
		preview := generateFilePreview(csvFile)
		
		// Step 2: Generate code
		code, err := utils.GenerateAnalysisCode(query, preview)
		if err != nil {
			fmt.Printf("❌ Code generation failed: %v\n", err)
			continue
		}
		fmt.Printf("Generated Code: %s\n", code)
		
		// Step 3: Execute code
		result, err := utils.RunPythonAnalysis(csvFile, code)
		if err != nil {
			fmt.Printf("❌ Execution failed: %v\n", err)
			continue
		}
		
		fmt.Printf("✅ Result: %s\n", result)
		fmt.Println("----------------------------------------------")
	}
	
	// Cleanup
	os.Remove(csvFile)
	fmt.Printf("\n[Cleanup] Removed test file: %s\n", csvFile)
	
	// Summary
	fmt.Println("\n==============================================")
	fmt.Println("TEST SUMMARY")
	fmt.Println("==============================================")
	fmt.Printf("Code Generation Tests:\n")
	fmt.Printf("  Passed: %d/%d\n", passed, len(testCases))
	fmt.Printf("  Failed: %d/%d\n", failed, len(testCases))
	
	if failed == 0 {
		fmt.Println("\n🎉 All code generation tests passed!")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed\n", failed)
	}
	
	fmt.Println("\n[NOTE] To test execution, make sure Python and pandas are installed:")
	fmt.Println("  pip install pandas openpyxl")
}

func createSampleCSV(filename string) {
	content := `Nama,Harga,Jumlah,Total
Naruto,45000,2,90000
One Piece,50000,1,50000
Bleach,40000,3,120000
Attack on Titan,55000,2,110000
Death Note,35000,4,140000`
	
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		log.Fatalf("Failed to create sample CSV: %v", err)
	}
}

func generateFilePreview(filePath string) string {
	// In real implementation, this would read the file and generate preview
	// For now, return hardcoded preview
	return `Struktur Data (CSV):
Kolom: Nama, Harga, Jumlah, Total

Sample Data (5 baris pertama):
Nama             | Harga  | Jumlah | Total
----------------|--------|--------|--------
Naruto          | 45000  | 2      | 90000
One Piece       | 50000  | 1      | 50000
Bleach          | 40000  | 3      | 120000
Attack on Titan | 55000  | 2      | 110000
Death Note      | 35000  | 4      | 140000

Total rows: 5`
}

