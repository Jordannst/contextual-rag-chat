import fs from 'fs';
import path from 'path';
import pdfParse from 'pdf-parse';

/**
 * Extract text from a file based on its extension
 * @param filePath - Path to the file
 * @returns Extracted text content
 */
export async function extractTextFromFile(filePath: string): Promise<string> {
  const fileExt = path.extname(filePath).toLowerCase();

  try {
    if (fileExt === '.pdf') {
      // Handle PDF files
      const dataBuffer = fs.readFileSync(filePath);
      const data = await pdfParse(dataBuffer);
      return data.text;
    } else if (fileExt === '.txt') {
      // Handle TXT files
      const text = fs.readFileSync(filePath, 'utf-8');
      return text;
    } else {
      throw new Error(`Unsupported file type: ${fileExt}`);
    }
  } catch (error) {
    throw new Error(`Error extracting text from file: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
}

