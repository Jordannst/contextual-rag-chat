package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	
	"backend/utils"
)

func main() {
	fmt.Println("==============================================")
	fmt.Println("CODE RUNNER TEST - Data Analyst Agent")
	fmt.Println("==============================================\n")
	
	// Cek apakah ada file CSV untuk test
	// Jika tidak ada, buat sample file
	testFile := "test_data.csv"
	if !fileExists(testFile) {
		fmt.Println("[Setup] Creating sample CSV file...")
		createSampleCSV(testFile)
		fmt.Printf("[Setup] Sample file created: %s\n\n", testFile)
	} else {
		fmt.Printf("[Setup] Using existing file: %s\n\n", testFile)
	}
	
	// Test cases
	tests := []struct {
		name string
		code string
	}{
		{
			name: "Get DataFrame shape",
			code: "print(df.shape)",
		},
		{
			name: "Get column names",
			code: "print(', '.join(df.columns))",
		},
		{
			name: "Calculate mean of Harga",
			code: "print(df['Harga'].mean())",
		},
		{
			name: "Calculate sum of Total",
			code: "print(df['Total'].sum())",
		},
		{
			name: "Count rows",
			code: "print(len(df))",
		},
		{
			name: "Filter data (Harga > 45000)",
			code: "print(len(df[df['Harga'] > 45000]))",
		},
		{
			name: "Get max Harga",
			code: "print(df['Harga'].max())",
		},
		{
			name: "Get min Total",
			code: "print(df['Total'].min())",
		},
		{
			name: "Describe Harga column",
			code: "print(df['Harga'].describe())",
		},
	}
	
	passed := 0
	failed := 0
	
	for i, test := range tests {
		fmt.Printf("[Test %d/%d] %s\n", i+1, len(tests), test.name)
		fmt.Printf("Code: %s\n", test.code)
		
		result, err := utils.RunPythonAnalysis(testFile, test.code)
		
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("✅ SUCCESS\n")
			fmt.Printf("Result:\n%s\n", result)
			passed++
		}
		
		fmt.Println("----------------------------------------------")
	}
	
	// Test dengan logging
	fmt.Println("\n[Test with Logging] Running analysis with full logs...")
	output, logs, err := utils.RunPythonAnalysisWithLogging(
		testFile,
		"print(f'Total items: {len(df)}, Average price: {df[\"Harga\"].mean()}')",
	)
	
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		fmt.Printf("✅ SUCCESS\n")
		fmt.Printf("Output: %s\n", output)
		fmt.Printf("Logs:\n%s\n", logs)
	}
	
	fmt.Println("==============================================")
	// Test error handling
	fmt.Println("\n[Error Handling Test] Testing invalid code...")
	_, err = utils.RunPythonAnalysis(testFile, "print(df['InvalidColumn'].mean())")
	if err != nil {
		fmt.Printf("✅ Correctly caught error: %v\n", err)
	} else {
		fmt.Println("❌ Should have failed but didn't!")
	}
	
	// Test validation
	fmt.Println("\n[Validation Test] Testing dangerous code detection...")
	dangerousCode := "import os; print(os.system('ls'))"
	err = utils.ValidatePythonCode(dangerousCode)
	if err != nil {
		fmt.Printf("✅ Correctly rejected dangerous code: %v\n", err)
	} else {
		fmt.Println("❌ Should have rejected but didn't!")
	}
	
	// Summary
	fmt.Println("\n==============================================")
	fmt.Println("TEST SUMMARY")
	fmt.Println("==============================================")
	fmt.Printf("Passed: %d/%d\n", passed, len(tests))
	fmt.Printf("Failed: %d/%d\n", failed, len(tests))
	
	if failed == 0 {
		fmt.Println("\n🎉 All tests passed!")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed\n", failed)
	}
	
	// Cleanup
	fmt.Printf("\n[Cleanup] Removing test file: %s\n", testFile)
	os.Remove(testFile)
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

func fileExists(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	_, err = os.Stat(absPath)
	return err == nil
}

