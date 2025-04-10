#!/bin/bash

# Simple script to create realistic test files for sortd testing
# Creates a variety of file types with proper headers/content

set -e

TEST_DIR="$HOME/sortd_test_files"
mkdir -p "$TEST_DIR/documents"
mkdir -p "$TEST_DIR/images"
mkdir -p "$TEST_DIR/archives"
mkdir -p "$TEST_DIR/media"
mkdir -p "$TEST_DIR/code"

echo "🚀 Creating test files for sortd in $TEST_DIR..."

# Create text files
echo -e "This is a simple text file.\nIt has multiple lines.\nCreated for sortd testing." > "$TEST_DIR/documents/readme.txt"
echo "Created text file: $TEST_DIR/documents/readme.txt"

echo -e "# Markdown Document\n\n## Introduction\n\nThis is a sample *markdown* file with **formatting**.\n\n- Item 1\n- Item 2" > "$TEST_DIR/documents/notes.md"
echo "Created markdown file: $TEST_DIR/documents/notes.md"

# Create a PDF file
cat > "$TEST_DIR/documents/report.pdf" << EOL
%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /Resources 4 0 R /MediaBox [0 0 612 792] /Contents 5 0 R >>
endobj
4 0 obj
<< /Font << /F1 6 0 R >> >>
endobj
5 0 obj
<< /Length 86 >>
stream
BT
/F1 12 Tf
50 700 Td
(Sample Document) Tj
0 -20 Td
(Created for testing sortd) Tj
ET
endstream
endobj
6 0 obj
<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>
endobj
xref
0 7
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000212 00000 n
0000000253 00000 n
0000000389 00000 n
trailer
<< /Size 7 /Root 1 0 R >>
startxref
456
%%EOF
EOL
echo "Created PDF file: $TEST_DIR/documents/report.pdf"

# Create a PNG file
echo -n -e '\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00\x90wS\xde\x00\x00\x00\x0cIDATx\x9cc```\x00\x00\x00\x04\x00\x01\xf6\x178U\x00\x00\x00\x00IEND\xaeB`\x82' > "$TEST_DIR/images/test.png"
echo "Created PNG file: $TEST_DIR/images/test.png"

# Create a JPEG file
printf "\xFF\xD8\xFF\xE0\x00\x10\x4A\x46\x49\x46\x00\x01\x01\x01\x00\x48\x00\x48\x00\x00\xFF\xDB\x00\x43\x00\xFF\xDB\x00\x43\x00\xFF\xC0\x00\x11\x08\x00\x01\x00\x01\x03\x01\x11\x00\x02\x11\x01\x03\x11\x01\xFF\xC4\x00\x1F\x00\x00\x01\x05\x01\x01\x01\x01\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0A\x0B\xFF\xC4\x00\xB5\x10\x00\x02\x01\x03\x03\x02\x04\x03\x05\x05\x04\x04\x00\x00\x01\x7D\x01\x02\x03\x00\x04\x11\x05\x12\x21\x31\x41\x06\x13\x51\x61\x07\x22\x71\x14\x32\x81\x91\xA1\x08\x23\x42\xB1\xC1\x15\x52\xD1\xF0\x24\x33\x62\x72\x82\x09\x0A\x16\x17\x18\x19\x1A\x25\x26\x27\x28\x29\x2A\x34\x35\x36\x37\x38\x39\x3A\x43\x44\x45\x46\x47\x48\x49\x4A\x53\x54\x55\x56\x57\x58\x59\x5A\x63\x64\x65\x66\x67\x68\x69\x6A\x73\x74\x75\x76\x77\x78\x79\x7A\x83\x84\x85\x86\x87\x88\x89\x8A\x92\x93\x94\x95\x96\x97\x98\x99\x9A\xA2\xA3\xA4\xA5\xA6\xA7\xA8\xA9\xAA\xB2\xB3\xB4\xB5\xB6\xB7\xB8\xB9\xBA\xC2\xC3\xC4\xC5\xC6\xC7\xC8\xC9\xCA\xD2\xD3\xD4\xD5\xD6\xD7\xD8\xD9\xDA\xE1\xE2\xE3\xE4\xE5\xE6\xE7\xE8\xE9\xEA\xF1\xF2\xF3\xF4\xF5\xF6\xF7\xF8\xF9\xFA\xFF\xDA\x00\x0C\x03\x01\x00\x02\x11\x03\x11\x00\x3F\x00\xFC\xFC\xA2\x8A\x28\x00\xA2\x8A\x28\x00\xFF\xD9" > "$TEST_DIR/images/photo.jpg"
echo "Created JPEG file: $TEST_DIR/images/photo.jpg"

# Create a GIF file
echo -n -e 'GIF89a\x01\x00\x01\x00\x80\x00\x00\xFF\xFF\xFF\x00\x00\x00!\xF9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;' > "$TEST_DIR/images/animation.gif"
echo "Created GIF file: $TEST_DIR/images/animation.gif"

# Create a ZIP file
echo -n -e 'PK\x03\x04\x0A\x00\x00\x00\x00\x00\x00\x00!\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00PK\x01\x02\x14\x00\x0A\x00\x00\x00\x00\x00\x00\x00!\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00PK\x05\x06\x00\x00\x00\x00\x01\x00\x01\x00.\x00\x00\x00\x1C\x00\x00\x00\x00\x00' > "$TEST_DIR/archives/data.zip"
echo "Created ZIP file: $TEST_DIR/archives/data.zip"

# Create a TAR.GZ file
echo -n -e '\x1F\x8B\x08\x00\x00\x00\x00\x00\x00\x03\xEB\xCE\xCC\xCD\xE3\x02\x00\xA1\x93\x2A\x37\x05\x00\x00\x00' > "$TEST_DIR/archives/backup.tar.gz"
echo "Created TAR.GZ file: $TEST_DIR/archives/backup.tar.gz"

# Create an MP3 file
echo -n -e 'ID3\x03\x00\x00\x00\x00\x00#TALB\x00\x00\x00\x0F\x00\x00\x03Sample Album\x00TCON\x00\x00\x00\x0B\x00\x00\x03Rock\x00\x00\xFF\xFB\x90\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00' > "$TEST_DIR/media/song.mp3"
echo "Created MP3 file: $TEST_DIR/media/song.mp3"

# Create an MP4 file
echo -n -e '\x00\x00\x00\x18\x66\x74\x79\x70\x6D\x70\x34\x32\x00\x00\x00\x00\x69\x73\x6F\x6D\x6D\x70\x34\x32\x00\x00\x00\x01\x6D\x6F\x6F\x76\x00\x00\x00\x6C\x6D\x76\x68\x64' > "$TEST_DIR/media/video.mp4"
echo "Created MP4 file: $TEST_DIR/media/video.mp4"

# Create a JSON file
cat > "$TEST_DIR/code/config.json" << EOL
{
    "name": "Test JSON",
    "version": "1.0.0",
    "description": "A sample JSON file for testing",
    "settings": {
        "enabled": true,
        "debug": false,
        "count": 42
    },
    "tags": ["test", "sortd", "example"]
}
EOL
echo "Created JSON file: $TEST_DIR/code/config.json"

# Create a YAML file
cat > "$TEST_DIR/code/settings.yaml" << EOL
# Sample YAML file for testing
name: Test YAML
version: 1.0.0
description: A sample YAML file for testing
settings:
  enabled: true
  debug: false
  count: 42
tags:
  - test
  - sortd
  - example
EOL
echo "Created YAML file: $TEST_DIR/code/settings.yaml"

# Create a Python script
cat > "$TEST_DIR/code/script.py" << EOL
#!/usr/bin/env python3
# Sample Python script for testing

class TestClass:
    def __init__(self, name):
        self.name = name
    
    def greet(self):
        return f"Hello, {self.name}!"

def main():
    test = TestClass("Sortd")
    print(test.greet())
    
if __name__ == "__main__":
    main()
EOL
chmod +x "$TEST_DIR/code/script.py"
echo "Created Python file: $TEST_DIR/code/script.py"

# Create a shell script
cat > "$TEST_DIR/code/run.sh" << EOL
#!/bin/bash
# Sample shell script for testing

echo "Testing sortd file organization tool"
for i in {1..5}; do
    echo "Item \$i"
done

exit 0
EOL
chmod +x "$TEST_DIR/code/run.sh"
echo "Created shell script: $TEST_DIR/code/run.sh"

# Create files with duplicate names to test collision handling
echo "Creating duplicate files for collision testing..."
cp "$TEST_DIR/documents/report.pdf" "$TEST_DIR/documents/report (copy).pdf"
echo "Created duplicate PDF: $TEST_DIR/documents/report (copy).pdf"

# Create files with special characters and spaces
echo "Creating files with special characters and spaces..."
echo "Content with special chars" > "$TEST_DIR/documents/Test & Document.txt"
echo "Created file with special chars: $TEST_DIR/documents/Test & Document.txt"

# Create a mixed-up Downloads folder to test sorting
mkdir -p "$TEST_DIR/Downloads_Mess"
cp "$TEST_DIR/documents/report.pdf" "$TEST_DIR/Downloads_Mess/"
cp "$TEST_DIR/images/photo.jpg" "$TEST_DIR/Downloads_Mess/"
cp "$TEST_DIR/archives/data.zip" "$TEST_DIR/Downloads_Mess/"
cp "$TEST_DIR/media/song.mp3" "$TEST_DIR/Downloads_Mess/"
cp "$TEST_DIR/code/script.py" "$TEST_DIR/Downloads_Mess/"
cp "$TEST_DIR/code/config.json" "$TEST_DIR/Downloads_Mess/"
echo "Created messy Downloads folder at: $TEST_DIR/Downloads_Mess"

echo "✅ All test files created successfully at $TEST_DIR"
echo "Use these files to test the sortd tool's organization capabilities"
