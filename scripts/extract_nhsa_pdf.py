#!/usr/bin/env python3
"""Extract all drug data from NHSA 2024 drug catalog PDF - fixed version."""

import pdfplumber
import json
import re

pdf_path = "F:/project/yaofang/data/nhsa_drug_catalog_2024.pdf"

def gbk_fix(text):
    """Fix garbled encoding from PDF extraction: latin-1 misinterpreted as GBK."""
    if not text:
        return ''
    try:
        return text.encode('latin-1').decode('gbk')
    except (UnicodeDecodeError, UnicodeEncodeError):
        return text

def is_header_or_classification(cells):
    """Check if row is a header or classification (not a drug entry)."""
    if not cells or len(cells) < 8:
        return True
    # Drug entries have a number in column 6
    col6 = gbk_fix(cells[6]) if len(cells) > 6 else ''
    if not col6 or not col6.strip().isdigit() and not col6.strip().replace('★','').replace('(','').replace(')','').isdigit():
        return True
    return False

def extract_drug_tables_from_pdf(pdf_path, start_page, end_page):
    """Extract drug data from a range of pages."""
    all_drugs = []
    # Track classification hierarchy
    hierarchy = {'l1': '', 'l2': '', 'l3': '', 'l4': '', 'l5': ''}

    with pdfplumber.open(pdf_path) as pdf:
        for page_num in range(start_page, min(end_page, len(pdf.pages))):
            page = pdf.pages[page_num]
            tables = page.extract_tables()

            for table in tables:
                for row in table:
                    if not row:
                        continue
                    # Pad
                    cells = list(row)
                    while len(cells) < 10:
                        cells.append(None)

                    # Fix encodings
                    cells_fixed = [gbk_fix(c) if c else '' for c in cells]

                    # Update classification hierarchy from columns 0-4
                    if cells_fixed[0].strip():
                        pass  # ATC code - hierarchy update tracked below
                    if cells_fixed[1].strip():
                        hierarchy['l1'] = cells_fixed[1].strip()
                    if cells_fixed[2].strip():
                        hierarchy['l2'] = cells_fixed[2].strip()
                    if cells_fixed[3].strip():
                        hierarchy['l3'] = cells_fixed[3].strip()
                    if cells_fixed[4].strip():
                        hierarchy['l4'] = cells_fixed[4].strip()

                    # Column 5: 甲乙类 (甲 or 乙)
                    # Column 6: drug number
                    # Column 7: drug name
                    # Column 8: dosage form
                    # Column 9: notes

                    col5 = cells_fixed[5].strip()  # 甲乙类
                    col6 = cells_fixed[6].strip()  # drug number
                    col7 = cells_fixed[7].strip()  # drug name
                    col8 = cells_fixed[8].strip()  # dosage form
                    col9 = cells_fixed[9].strip()  # notes

                    # Skip header rows, classification rows, empty rows
                    if not col7:
                        continue
                    # Skip the header row itself
                    if col7 in ('药品名称', 'ҩƷ����'):
                        continue
                    # Skip rows that are classification descriptions not drugs
                    if not col6 or (not any(c in col5 for c in ['甲', '乙', '��'])):
                        # Might still be a drug without clear marker - check if col6 has a number
                        cleaned = col6.replace('(','').replace(')','').replace('��','').strip()
                        if not cleaned.isdigit():
                            continue

                    # Parse insurance category
                    is_class_a = '甲' in col5 or '��' in col5
                    is_class_b = '乙' in col5 or '��' in col5
                    is_reference = '★' in col5 or '��' in col5 or '��' in col6

                    # Clean drug number
                    drug_num = col6.replace('★', '').replace('��', '').replace('(', '').replace(')', '').strip()

                    drug = {
                        'drug_num': drug_num,
                        'drug_name': col7,
                        'dosage_form': col8,
                        'is_class_a': is_class_a,
                        'is_class_b': is_class_b,
                        'is_reference': is_reference,
                        'notes': col9,
                        'class_l1': hierarchy['l1'],
                        'class_l2': hierarchy['l2'],
                        'class_l3': hierarchy['l3'],
                        'class_l4': hierarchy['l4'],
                    }
                    all_drugs.append(drug)

    return all_drugs

# Extract western medicine (pages 8-81, 0-indexed: 7-80)
print("Extracting western medicine...")
western = extract_drug_tables_from_pdf(pdf_path, 7, 80)
print(f"  {len(western)} entries")

# Extract TCM (pages 82-127, 0-indexed: 81-126)
print("Extracting TCM...")
tcm = extract_drug_tables_from_pdf(pdf_path, 81, 126)
print(f"  {len(tcm)} entries")

all_drugs = western + tcm
print(f"Total: {len(all_drugs)}")

# Deduplicate by (name, form)
seen = set()
unique = []
for d in all_drugs:
    key = (d['drug_name'], d['dosage_form'])
    if key not in seen:
        seen.add(key)
        unique.append(d)

print(f"Unique: {len(unique)}")
print(f"  Class A: {sum(1 for d in unique if d['is_class_a'])}")
print(f"  Class B: {sum(1 for d in unique if d['is_class_b'])}")
print(f"  References: {sum(1 for d in unique if d['is_reference'])}")

# Save
with open('F:/project/yaofang/data/nhsa_drugs_unique.json', 'w', encoding='utf-8') as f:
    json.dump(unique, f, ensure_ascii=False, indent=2)

# Print some samples to verify encoding
for d in unique[10:15]:
    cls = '甲' if d['is_class_a'] else ('乙' if d['is_class_b'] else '?')
    print(f"  [{cls}] {d['drug_name'][:30]} | {d['dosage_form'][:20]} | {d['class_l2'][:20]}")

print("Done!")
