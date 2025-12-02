#!/usr/bin/env python3
"""
Code Interpreter untuk Data Analyst Agent
Menjalankan kode Python pada dataframe dari file CSV/Excel
"""

import sys
import io
import os
import json
import base64
import re
import pandas as pd
import matplotlib
matplotlib.use('Agg')  # Non-interactive backend
import matplotlib.pyplot as plt
import seaborn as sns


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
    Cegah penggunaan fungsi berbahaya dan auto-fix common issues
    """
    # Auto-fix: Replace plt.show() dengan show_chart() (case-insensitive)
    # Handle berbagai variasi: plt.show(), plt.show( ), plt.show(block=True), dll
    # Pattern untuk match plt.show() dengan berbagai variasi
    # Match: plt.show() atau plt.show(...) dengan parameter apapun
    plt_show_pattern = re.compile(r'plt\.show\s*\([^)]*\)', re.IGNORECASE)
    code = plt_show_pattern.sub('show_chart()', code)
    
    # Daftar keywords berbahaya yang tidak boleh ada
    # Note: matplotlib dan seaborn sudah diimport, jadi tidak perlu block
    # plt.show() sudah di-handle dengan auto-replace di atas, jadi tidak perlu di-forbidden
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
    Tangkap output dari print statements dan chart data
    """
    # Siapkan environment untuk exec
    # Berikan akses ke pandas dan numpy (common libraries untuk analisis)
    import numpy as np
    
    # Helper function untuk menampilkan chart
    def show_chart():
        """
        Simpan plot saat ini ke buffer memory, encode ke Base64,
        dan print dengan format khusus untuk parsing Go.
        Setelah mencetak, tutup figure agar tidak terjadi duplikasi.
        """
        # Cek apakah ada figure aktif
        if len(plt.get_fignums()) == 0:
            # Tidak ada figure, tidak perlu melakukan apa-apa
            return
        
        # Simpan plot ke buffer memory
        buffer = io.BytesIO()
        plt.savefig(buffer, format='png', dpi=100, bbox_inches='tight')
        buffer.seek(0)
        
        # Encode ke Base64 string
        chart_base64 = base64.b64encode(buffer.read()).decode('utf-8')
        
        # Print dengan format khusus untuk parsing Go
        print(f"[CHART_DATA:{chart_base64}]")
        
        # Tutup semua figure setelah mencetak agar tidak terjadi duplikasi
        # Ini penting untuk auto-flush: jika AI sudah panggil show_chart(), 
        # figure sudah ditutup, jadi auto-flush tidak akan duplikat
        plt.close('all')
        buffer.close()
    
    exec_globals = {
        "df": df,
        "pd": pd,
        "np": np,
        "plt": plt,
        "sns": sns,
        "show_chart": show_chart,
        "__builtins__": __builtins__,
    }
    
    # Capture stdout
    old_stdout = sys.stdout
    sys.stdout = captured_output = io.StringIO()
    
    try:
        # Jalankan kode
        exec(code, exec_globals)
        
        # AUTO-FLUSH: Cek apakah ada figure matplotlib yang aktif setelah exec selesai
        # Ini memastikan grafik tetap terkirim meskipun AI lupa memanggil show_chart()
        active_figures = plt.get_fignums()
        if len(active_figures) > 0:
            # Ada figure aktif yang belum di-display
            # Panggil show_chart() secara otomatis
            sys.stderr.write(f"[CodeInterpreter] Auto-flush: Found {len(active_figures)} active figure(s), automatically calling show_chart()\n")
            show_chart()
            # Note: show_chart() sudah menutup figure, jadi tidak perlu close lagi
        
        # Ambil output
        output = captured_output.getvalue()
        return output.strip()
    
    finally:
        # Restore stdout
        sys.stdout = old_stdout
        # Cleanup: close any remaining figures (safety net, biasanya sudah ditutup oleh show_chart)
        plt.close('all')


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
        original_code = code_to_run
        sanitized_code = sanitize_code(code_to_run)
        
        # Log jika ada perubahan (auto-fix)
        if original_code != sanitized_code:
            sys.stderr.write(f"[CodeInterpreter] Auto-fixed code (replaced plt.show() with show_chart())\n")
        
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

