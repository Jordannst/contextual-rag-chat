#!/usr/bin/env python3
"""
Code Interpreter untuk Data Analyst Agent
Menjalankan kode Python pada dataframe dari file CSV/Excel
"""

import sys
import io
import os
import json
import pandas as pd


def usage_and_exit():
    sys.stderr.write("Usage: python code_interpreter.py <file_path> <code_to_run>\n")
    sys.stderr.write("Example: python code_interpreter.py data.csv \"print(df['Harga'].mean())\"\n")
    sys.exit(1)


def load_data(file_path: str) -> pd.DataFrame:
    """Load CSV atau Excel file ke pandas DataFrame"""
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"File tidak ditemukan: {file_path}")
    
    ext = os.path.splitext(file_path)[1].lower()
    
    try:
        if ext == ".csv":
            df = pd.read_csv(file_path)
        elif ext in [".xlsx", ".xls"]:
            df = pd.read_excel(file_path)
        else:
            raise ValueError(f"Format file tidak didukung: {ext}. Gunakan .csv, .xlsx, atau .xls")
        
        # Bersihkan NaN
        df = df.fillna("")
        return df
    except Exception as e:
        raise RuntimeError(f"Gagal membaca file: {e}")


def sanitize_code(code: str) -> str:
    """
    Sanitasi kode sederhana untuk keamanan dasar
    Cegah penggunaan fungsi berbahaya
    """
    # Daftar keywords berbahaya yang tidak boleh ada
    dangerous_keywords = [
        "import os",
        "import sys",
        "import subprocess",
        "__import__",
        "eval(",
        "exec(",  # Nested exec
        "compile(",
        "open(",  # File operations
        "file(",
        "input(",
        "raw_input(",
    ]
    
    code_lower = code.lower()
    for keyword in dangerous_keywords:
        if keyword in code_lower:
            raise ValueError(
                f"Kode tidak diizinkan: mengandung '{keyword}'. "
                f"Hanya operasi pandas dan matematika yang diperbolehkan."
            )
    
    return code


def run_code(df: pd.DataFrame, code: str) -> str:
    """
    Jalankan kode Python dengan dataframe yang tersedia
    Tangkap output dari print statements
    """
    # Siapkan environment untuk exec
    # Berikan akses ke pandas dan numpy (common libraries untuk analisis)
    import numpy as np
    
    exec_globals = {
        "df": df,
        "pd": pd,
        "np": np,
        "__builtins__": __builtins__,
    }
    
    # Capture stdout
    old_stdout = sys.stdout
    sys.stdout = captured_output = io.StringIO()
    
    try:
        # Jalankan kode
        exec(code, exec_globals)
        
        # Ambil output
        output = captured_output.getvalue()
        return output.strip()
    
    finally:
        # Restore stdout
        sys.stdout = old_stdout


def main():
    if len(sys.argv) < 3:
        usage_and_exit()
    
    file_path = sys.argv[1]
    code_to_run = sys.argv[2]
    
    try:
        # Load data
        sys.stderr.write(f"[CodeInterpreter] Loading file: {file_path}\n")
        df = load_data(file_path)
        sys.stderr.write(f"[CodeInterpreter] Data loaded: {df.shape[0]} rows, {df.shape[1]} columns\n")
        sys.stderr.write(f"[CodeInterpreter] Columns: {', '.join(df.columns.tolist())}\n")
        
        # Sanitasi kode
        sys.stderr.write(f"[CodeInterpreter] Sanitizing code...\n")
        sanitized_code = sanitize_code(code_to_run)
        
        # Jalankan kode
        sys.stderr.write(f"[CodeInterpreter] Executing code:\n{sanitized_code}\n")
        result = run_code(df, sanitized_code)
        
        sys.stderr.write(f"[CodeInterpreter] Execution successful\n")
        
        # Output hasil ke stdout (UTF-8)
        if result:
            sys.stdout.buffer.write(result.encode("utf-8", errors="ignore"))
        else:
            # Jika tidak ada output, kirim pesan kosong
            sys.stdout.buffer.write(b"")
    
    except SyntaxError as e:
        error_msg = f"SyntaxError: {e}\nKode Python tidak valid."
        sys.stderr.write(json.dumps({"error": error_msg}) + "\n")
        sys.exit(1)
    
    except (KeyError, ValueError, TypeError, AttributeError) as e:
        # Error umum saat analisis data
        error_msg = f"{type(e).__name__}: {e}"
        sys.stderr.write(json.dumps({"error": error_msg}) + "\n")
        sys.exit(1)
    
    except Exception as e:
        error_msg = f"Error: {e}"
        sys.stderr.write(json.dumps({"error": error_msg}) + "\n")
        sys.exit(1)


if __name__ == "__main__":
    main()

