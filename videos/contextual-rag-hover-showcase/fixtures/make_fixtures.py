"""Generate the fictional Nara Studio demo fixtures (PDF + CSV).

Run:  python3 make_fixtures.py   (writes Q3_Report.pdf and Monthly_Revenue.csv next to this file)
All figures are fictional showcase data.
"""
import os

from reportlab.lib import colors
from reportlab.lib.pagesizes import A4
from reportlab.pdfgen import canvas

HERE = os.path.dirname(os.path.abspath(__file__))

MONTHS = [("July", 120_000_000), ("August", 148_000_000), ("September", 186_000_000)]
TOTAL = sum(v for _, v in MONTHS)
GROWTH = (MONTHS[2][1] - MONTHS[0][1]) / MONTHS[0][1] * 100
assert TOTAL == 454_000_000
assert round(GROWTH) == 55 and abs(GROWTH - 55) < 1e-9

INK = colors.HexColor("#111827")
MUTED = colors.HexColor("#6B7280")
LINE = colors.HexColor("#E5E7EB")
ACCENT = colors.HexColor("#3B5BDB")
TINT = colors.HexColor("#EEF2FF")


def fmt(v: int) -> str:
    return f"{v:,}"


def make_pdf(path: str) -> None:
    w, h = A4
    c = canvas.Canvas(path, pagesize=A4)
    c.setTitle("Nara Studio — Q3 Revenue Review")
    c.setAuthor("Nara Studio (fictional demo data)")
    m = 56  # side margin

    # Header band
    c.setFillColor(ACCENT)
    c.rect(0, h - 10, w, 10, stroke=0, fill=1)
    c.setFillColor(MUTED)
    c.setFont("Helvetica-Bold", 9)
    c.drawString(m, h - 48, "NARA STUDIO  ·  QUARTERLY REPORT")
    # Fictional demo data tag
    tag = "Fictional demo data"
    c.setFont("Helvetica", 8.5)
    tw = c.stringWidth(tag, "Helvetica", 8.5)
    c.setFillColor(colors.HexColor("#FEF3C7"))
    c.roundRect(w - m - tw - 16, h - 54, tw + 16, 18, 9, stroke=0, fill=1)
    c.setFillColor(colors.HexColor("#92400E"))
    c.drawString(w - m - tw - 8, h - 48, tag)

    # Title
    c.setFillColor(INK)
    c.setFont("Helvetica-Bold", 24)
    c.drawString(m, h - 86, "Nara Studio — Q3 Revenue Review")
    c.setFillColor(MUTED)
    c.setFont("Helvetica", 11)
    c.drawString(m, h - 106, "Reporting period: July – September")

    c.setStrokeColor(LINE)
    c.setLineWidth(1)
    c.line(m, h - 122, w - m, h - 122)

    # Summary
    c.setFillColor(INK)
    c.setFont("Helvetica-Bold", 12)
    c.drawString(m, h - 148, "Summary")
    c.setFont("Helvetica", 11.5)
    summary = [
        "Q3 revenue totaled IDR 454,000,000. Monthly revenue increased each month",
        "of the quarter. September revenue reached IDR 186,000,000, up 55%",
        "compared with July.",
    ]
    y = h - 168
    for line in summary:
        c.drawString(m, y, line)
        y -= 17

    # Revenue table
    y -= 16
    c.setFont("Helvetica-Bold", 12)
    c.drawString(m, y, "Monthly revenue")
    y -= 14
    tx0, tx1 = m, w - m
    row_h = 26
    # header row
    c.setFillColor(colors.HexColor("#F3F4F6"))
    c.rect(tx0, y - row_h, tx1 - tx0, row_h, stroke=0, fill=1)
    c.setFillColor(MUTED)
    c.setFont("Helvetica-Bold", 10)
    c.drawString(tx0 + 12, y - 17, "Month")
    c.drawRightString(tx1 - 12, y - 17, "Revenue (IDR)")
    y -= row_h
    c.setFont("Helvetica", 11.5)
    for name, val in MONTHS:
        c.setFillColor(INK)
        c.drawString(tx0 + 12, y - 18, name)
        c.drawRightString(tx1 - 12, y - 18, fmt(val))
        c.setStrokeColor(LINE)
        c.line(tx0, y - row_h, tx1, y - row_h)
        y -= row_h
    # total row
    c.setFillColor(TINT)
    c.rect(tx0, y - row_h, tx1 - tx0, row_h, stroke=0, fill=1)
    c.setFillColor(INK)
    c.setFont("Helvetica-Bold", 11.5)
    c.drawString(tx0 + 12, y - 18, "Q3 total")
    c.drawRightString(tx1 - 12, y - 18, fmt(TOTAL))
    y -= row_h

    # Key results
    y -= 30
    c.setFont("Helvetica-Bold", 12)
    c.drawString(m, y, "Key Results")
    y -= 22
    results = [
        ("Q3 total revenue", "IDR 454,000,000"),
        ("September revenue", "IDR 186,000,000"),
        ("September vs. July", "+55%"),
    ]
    for label, value in results:
        c.setFillColor(ACCENT)
        c.circle(m + 4, y + 4, 2.6, stroke=0, fill=1)
        c.setFillColor(INK)
        c.setFont("Helvetica", 11.5)
        c.drawString(m + 16, y, label + ":")
        c.setFont("Helvetica-Bold", 11.5)
        c.drawString(m + 16 + c.stringWidth(label + ": ", "Helvetica", 11.5), y, value)
        y -= 20

    # Footer
    c.setFillColor(MUTED)
    c.setFont("Helvetica", 8.5)
    c.drawString(m, 40, "Nara Studio is a fictional company. All figures are demo data created for a product showcase.")
    c.drawRightString(w - m, 40, "Page 1 of 1")
    c.showPage()
    c.save()


def make_csv(path: str) -> None:
    with open(path, "w", newline="") as f:
        f.write("month,revenue_idr\n")
        for name, val in MONTHS:
            f.write(f"{name},{val}\n")


if __name__ == "__main__":
    make_pdf(os.path.join(HERE, "Q3_Report.pdf"))
    make_csv(os.path.join(HERE, "Monthly_Revenue.csv"))
    print("total", TOTAL, "growth", GROWTH)
