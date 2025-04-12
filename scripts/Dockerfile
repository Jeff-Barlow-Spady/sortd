# Dockerfile for Sortd Testing Environment
FROM golang:1.23-bullseye

# Install required tools
RUN apt-get update && apt-get install -y \
    gcc-mingw-w64 \
    zip \
    unzip \
    git \
    curl \
    python3 \
    python3-pip \
    jq \
    vim \
    tmux \
    && rm -rf /var/lib/apt/lists/*

# Install AWS CLI for S3 bucket testing
RUN pip3 install awscli boto3

# Install Go cross compiler for Windows
RUN go install github.com/mitchellh/gox@latest

# Install additional Go tools
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install golang.org/x/lint/golint@latest

# Set up directories
WORKDIR /app
RUN mkdir -p /app/test_sandbox/mock_filesystem/Downloads \
    /app/test_sandbox/sortd-config/workflows \
    /app/test_sandbox/logs

# Copy build scripts (will be overridden by volume mount in development)
COPY scripts/build_*.sh /app/scripts/
RUN chmod +x /app/scripts/build_*.sh

# Set up persistent volumes
VOLUME ["/app/src", "/app/test_sandbox"]

# Environment variables
ENV PATH="/app/bin:${PATH}" \
    GOPATH="/go" \
    CGO_ENABLED=1

# Create build scripts for different platforms
RUN echo '#!/bin/bash\ncd /app/src && go build -o /app/bin/sortd ./cmd/sortd/' > /app/scripts/build_linux.sh && \
    echo '#!/bin/bash\ncd /app/src && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o /app/bin/sortd.exe ./cmd/sortd/' > /app/scripts/build_windows.sh && \
    chmod +x /app/scripts/build_*.sh

# Create helper script to prepare the test environment
RUN echo '#!/bin/bash\n\
# Create test files\n\
DOWNLOADS="/app/test_sandbox/mock_filesystem/Downloads"\n\
mkdir -p "$DOWNLOADS"\n\
mkdir -p "$DOWNLOADS/Documents" "$DOWNLOADS/Images" "$DOWNLOADS/Music" "$DOWNLOADS/Videos" "$DOWNLOADS/Archives"\n\
\n\
# Create document files\n\
for i in {1..5}; do\n\
    echo "This is test document $i" > "$DOWNLOADS/test_doc_$i.txt"\n\
    touch "$DOWNLOADS/test_doc_$i.pdf"\n\
    touch "$DOWNLOADS/invoice_$i.pdf"\n\
done\n\
\n\
# Create image files\n\
for i in {1..5}; do\n\
    touch "$DOWNLOADS/test_image_$i.jpg"\n\
    touch "$DOWNLOADS/test_image_$i.png"\n\
    touch "$DOWNLOADS/vacation_pic_$i.jpg"\n\
done\n\
\n\
# Create audio files\n\
for i in {1..3}; do\n\
    touch "$DOWNLOADS/test_song_$i.mp3"\n\
    touch "$DOWNLOADS/test_audio_$i.wav"\n\
done\n\
\n\
# Create video files\n\
for i in {1..3}; do\n\
    touch "$DOWNLOADS/test_video_$i.mp4"\n\
    touch "$DOWNLOADS/movie_$i.mkv"\n\
done\n\
\n\
# Create archive files\n\
for i in {1..3}; do\n\
    touch "$DOWNLOADS/test_archive_$i.zip"\n\
    touch "$DOWNLOADS/backup_$i.tar.gz"\n\
done\n\
\n\
# Create config files\n\
CONFIG_DIR="/app/test_sandbox/sortd-config"\n\
WORKFLOWS_DIR="$CONFIG_DIR/workflows"\n\
mkdir -p "$WORKFLOWS_DIR"\n\
\n\
# Create config.yaml\n\
cat > "$CONFIG_DIR/config.yaml" << EOF\n\
# Sortd Configuration for Test Environment\n\
version: 1\n\
\n\
# Global Settings\n\
settings:\n\
  dry_run: false\n\
  create_dirs: true\n\
  collision_strategy: "rename"\n\
  confirm_operations: false\n\
\n\
# Sorting patterns\n\
patterns:\n\
  - match: "*.{jpg,jpeg,png,gif,bmp}"\n\
    target: "Images/"\n\
  - match: "*.{doc,docx,pdf,txt,md,rtf}"\n\
    target: "Documents/"\n\
  - match: "*.{mp3,wav,flac,ogg,m4a}"\n\
    target: "Music/"\n\
  - match: "*.{mp4,mkv,avi,mov,wmv}"\n\
    target: "Videos/"\n\
  - match: "*.{zip,tar,gz,rar,7z}"\n\
    target: "Archives/"\n\
  - match: "invoice_*.pdf"\n\
    target: "Documents/Invoices/"\n\
\n\
# Watch directories\n\
watch_directories:\n\
  - "/app/test_sandbox/mock_filesystem/Downloads"\n\
EOF\n\
\n\
# Create workflows\n\
cat > "$WORKFLOWS_DIR/document_processor.yaml" << EOF\n\
id: "document-processor"\n\
name: "Document Processor"\n\
description: "Process documents based on content and type"\n\
enabled: true\n\
priority: 5\n\
\n\
trigger:\n\
  type: "FileCreated"\n\
  pattern: "*.{pdf,txt,doc,docx}"\n\
\n\
conditions:\n\
  - type: "FileCondition"\n\
    field: "name"\n\
    operator: "Contains"\n\
    value: "invoice"\n\
    caseSensitive: false\n\
\n\
actions:\n\
  - type: "MoveAction"\n\
    target: "/app/test_sandbox/mock_filesystem/Downloads/Documents/Invoices"\n\
    options:\n\
      createTargetDir: "true"\n\
EOF\n\
\n\
cat > "$WORKFLOWS_DIR/image_sorter.yaml" << EOF\n\
id: "image-sorter"\n\
name: "Image Sorter"\n\
description: "Sort images into appropriate folders"\n\
enabled: true\n\
priority: 4\n\
\n\
trigger:\n\
  type: "FileCreated"\n\
  pattern: "*.{jpg,jpeg,png,gif}"\n\
\n\
conditions:\n\
  - type: "FileCondition"\n\
    field: "name"\n\
    operator: "Contains"\n\
    value: "vacation"\n\
    caseSensitive: false\n\
\n\
actions:\n\
  - type: "MoveAction"\n\
    target: "/app/test_sandbox/mock_filesystem/Downloads/Images/Vacation"\n\
    options:\n\
      createTargetDir: "true"\n\
EOF\n\
\n\
# S3 integration example workflow\n\
cat > "$WORKFLOWS_DIR/s3_upload.yaml" << EOF\n\
id: "s3-upload"\n\
name: "S3 Upload"\n\
description: "Upload documents to S3"\n\
enabled: true\n\
priority: 3\n\
\n\
trigger:\n\
  type: "FileCreated"\n\
  pattern: "*.pdf"\n\
\n\
conditions:\n\
  - type: "FileCondition"\n\
    field: "size"\n\
    operator: "LessThan"\n\
    value: "10"\n\
    valueUnit: "MB"\n\
\n\
actions:\n\
  - type: "CommandAction"\n\
    command: "aws s3 cp \"{{ .FilePath }}\" \"s3://sortd-test-bucket/uploads/{{ .FileName }}\" --region us-east-1"\n\
    options:\n\
      runAsynchronously: false\n\
      shell: "/bin/bash"\n\
EOF\n\
\n\
echo "Test environment prepared successfully."\n\
' > /app/scripts/prepare_test_env.sh && chmod +x /app/scripts/prepare_test_env.sh

# Create run wrapper script
RUN echo '#!/bin/bash\n\
CONFIG_DIR="/app/test_sandbox/sortd-config"\n\
\n\
function print_banner {\n\
    echo -e "\033[0;35m"\n\
    echo "╔═════════════════════════════════════════════════════════╗"\n\
    echo "║                                                         ║"\n\
    echo "║   SORTD DOCKER TEST ENVIRONMENT                         ║"\n\
    echo "║   Let Chaos Sort Itself Out! 🗂️                         ║"\n\
    echo "║                                                         ║"\n\
    echo "╚═════════════════════════════════════════════════════════╝"\n\
    echo -e "\033[0m"\n\
}\n\
\n\
print_banner\n\
\n\
echo -e "\033[0;32m"\n\
echo "Available commands:"\n\
echo "  build_linux - Build Sortd for Linux"\n\
echo "  build_windows - Build Sortd for Windows"\n\
echo "  prepare_env - Prepare test environment"\n\
echo "  run [args] - Run Sortd with arguments"\n\
echo "  run_organize - Test basic organization"\n\
echo "  run_workflow [id] - Run specific workflow"\n\
echo -e "\033[0m"\n\
\n\
function run_organize {\n\
    /app/bin/sortd organize --config=$CONFIG_DIR --dir=/app/test_sandbox/mock_filesystem/Downloads --non-interactive\n\
}\n\
\n\
function run_workflow {\n\
    /app/bin/sortd workflow run --config=$CONFIG_DIR --id=$1 --non-interactive\n\
}\n\
\n\
# Set up PATH to include binaries\n\
export PATH="/app/bin:$PATH"\n\
' > /app/scripts/sortd_wrapper.sh && chmod +x /app/scripts/sortd_wrapper.sh

# Create entrypoint script
RUN echo '#!/bin/bash\n\
if [ "$1" = "prepare" ]; then\n\
    /app/scripts/prepare_test_env.sh\n\
elif [ "$1" = "build_linux" ]; then\n\
    /app/scripts/build_linux.sh\n\
elif [ "$1" = "build_windows" ]; then\n\
    /app/scripts/build_windows.sh\n\
elif [ "$1" = "run_organize" ]; then\n\
    /app/bin/sortd organize --config=/app/test_sandbox/sortd-config --dir=/app/test_sandbox/mock_filesystem/Downloads --non-interactive\n\
elif [ "$1" = "run_workflow" ] && [ ! -z "$2" ]; then\n\
    /app/bin/sortd workflow run --config=/app/test_sandbox/sortd-config --id=$2 --non-interactive\n\
elif [ "$1" = "run" ] && [ ! -z "$2" ]; then\n\
    shift\n\
    /app/bin/sortd "$@"\n\
else\n\
    # Interactive shell with wrapper loaded\n\
    source /app/scripts/sortd_wrapper.sh\n\
    exec /bin/bash\n\
fi\n\
' > /app/entrypoint.sh && chmod +x /app/entrypoint.sh

ENTRYPOINT ["/app/entrypoint.sh"]