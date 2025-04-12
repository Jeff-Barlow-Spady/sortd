# Testing Strategy for Content Analysis and Learning Integration

This document outlines the comprehensive testing approach for the content analysis and learning system integration with the organization engine.

## 1. Unit Testing

### 1.1 Content Analysis Testing

- **Content Signature Generation Tests**
  - Test text signature generation with various text files
  - Test image signature generation with various image formats
  - Test document signature generation with PDFs and Office documents
  - Test binary signature generation with different binary files
  - Test empty and very large files

- **Content Similarity Tests**
  - Test similarity detection between identical files
  - Test similarity detection between similar but not identical files
  - Test similarity detection between completely different files
  - Test similarity thresholds and scoring

- **Content Relationship Tests**
  - Test relationship detection between files
  - Test relationship type classification
  - Test relationship sorting by similarity score
  - Test performance with large numbers of files

- **Content Group Tests**
  - Test group creation
  - Test membership management
  - Test group metadata
  - Test persistence of groups

### 1.2 Classification Tests

- **File Classification Tests**
  - Test classification of files by extension
  - Test classification of files by name patterns
  - Test classification of files by MIME type
  - Test classification of files by content
  - Test confidence scoring
  - Test classification thresholds

- **Content-Enhanced Classification Tests**
  - Test how content analysis improves classification accuracy
  - Test classification inheritance through content relationships
  - Test content group based classification
  - Test confidence adjustment with content analysis

### 1.3 Integration Layer Tests

- **OrganizeEngineIntegration Tests**
  - Test pattern generation from classifications
  - Test file enrichment with classification data
  - Test destination suggestion based on content
  - Test operation tracking

- **EngineAdapter Tests**
  - Test integration with learning engine
  - Test content-enhanced organization
  - Test operation tracking
  - Test pattern forwarding

## 2. Integration Testing

### 2.1 Engine Interaction Testing

- **Learning to Organization Flow**
  - Test classification-based organization
  - Test content relationship discovery during organization
  - Test content signature generation during organization
  - Test operation recording

- **Full System Integration**
  - Test daemon initialization of learning engine
  - Test file watcher integration with learning
  - Test workflow integration with learning
  - Test manual and automatic organization with learning

### 2.2 Database Tests

- **Repository Tests**
  - Test all repository methods
  - Test transaction handling
  - Test concurrent access
  - Test error conditions

- **Persistence Tests**
  - Test data persistence across restarts
  - Test schema migration (if implemented)
  - Test performance with large datasets

## 3. Docker Testing Environment

### 3.1 Docker Test Container Setup

```bash
# Create Docker test container
docker build -t sortd-test -f Dockerfile.test .

# Run tests in container
docker run --rm -v $(pwd):/app sortd-test go test -v ./...
```

### 3.2 Test Scenarios in Docker

#### Basic Content Analysis Tests

```bash
# Test content analysis with various files
docker run --rm -v $(pwd):/app sortd-test go run ./cmd/test-content-analysis \
  --files=/test-files \
  --report=/reports/content-analysis.json
```

#### Classification Accuracy Tests

```bash
# Test classification with standard file sets
docker run --rm -v $(pwd):/app sortd-test go run ./cmd/test-classification \
  --files=/test-files \
  --expected=/test-files/expected-classifications.json \
  --report=/reports/classification-accuracy.json
```

#### Organization Tests

```bash
# Test organization with content analysis
docker run --rm -v $(pwd):/app sortd-test go run ./cmd/test-organize \
  --source=/test-files/source \
  --expected=/test-files/expected-organization \
  --report=/reports/organization-accuracy.json
```

## 4. Performance Testing

### 4.1 Benchmarks

- **Content Analysis Benchmarks**
  - Benchmark signature generation for different file types
  - Benchmark similarity calculation
  - Benchmark relationship discovery

- **Classification Benchmarks**
  - Benchmark classification time
  - Benchmark confidence calculation
  - Benchmark classification with different criteria

- **Integration Benchmarks**
  - Benchmark end-to-end organization with learning
  - Benchmark database operations

### 4.2 Scalability Tests

- **Large Dataset Tests**
  - Test with 10,000+ files
  - Test with 1,000+ relationships
  - Test with 100+ classifications

- **Memory Usage Tests**
  - Test memory consumption during analysis
  - Test memory leaks with long-running processes
  - Test garbage collection behavior

## 5. Test Data Preparation

### 5.1 Test File Sets

- **Standard Test Files**
  - Text files with different content patterns
  - Images with various formats and content
  - Documents with different structures
  - Binary files with different formats
  - Files with specific naming patterns

- **Relationship Test Sets**
  - Similar text content in different formats
  - Similar images with different resolutions
  - Document revisions and versions
  - Related binary files

- **Classification Test Sets**
  - Files matching specific classification criteria
  - Edge cases for classification
  - Files with ambiguous classification

### 5.2 Expected Results

- **Classification Expectations**
  - JSON files mapping file paths to expected classifications

- **Organization Expectations**
  - Directory structures representing expected organization

- **Relationship Expectations**
  - JSON files describing expected file relationships

## 6. Continuous Integration Setup

### 6.1 CI Pipeline

```yaml
# .github/workflows/content-analysis-tests.yml
name: Content Analysis Tests

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.20'
    - name: Unit Tests
      run: go test -v ./internal/patterns/learning/...
    - name: Integration Tests
      run: go test -v ./internal/patterns/learning/integration/...
    - name: Docker Tests
      run: |
        docker build -t sortd-test -f Dockerfile.test .
        docker run --rm sortd-test go run ./cmd/test-content-analysis
```

## 7. Manual Testing Checklist

### 7.1 Content Analysis Validation

- [ ] Verify content signatures reflect actual content
- [ ] Verify similarity scores match intuitive similarity
- [ ] Verify relationship types are accurately detected
- [ ] Verify content groups contain logically related files

### 7.2 Classification Validation

- [ ] Verify classifications match expected file categories
- [ ] Verify confidence scores reflect classification certainty
- [ ] Verify classification criteria work as expected
- [ ] Verify content enhances classification accuracy

### 7.3 Integration Validation

- [ ] Verify content analysis improves organization
- [ ] Verify operation tracking records all activities
- [ ] Verify daemon initialization works correctly
- [ ] Verify complete system behavior is as expected

## 8. Bug Reporting and Tracking

All bugs found during testing should be documented with:

1. **Bug Description**: Clear description of the issue
2. **Reproduction Steps**: Exact steps to reproduce the bug
3. **Expected Behavior**: What should happen
4. **Actual Behavior**: What actually happens
5. **Test Data**: Files or data that trigger the bug
6. **Environment**: Docker, native, etc.
7. **Severity**: Critical, Major, Minor, Cosmetic

## 9. Test Implementation Timeline

| Week | Focus Area | Key Tasks |
|------|------------|-----------|
| 1    | Unit Tests | Implement core content analysis tests and classification tests |
| 2    | Integration Tests | Implement engine integration tests and database tests |
| 3    | Docker Environment | Set up Docker testing environment and scenarios |
| 4    | Performance Tests | Implement benchmarks and scalability tests |

## 10. Getting Started with Testing

To start testing the content analysis system:

1. Clone the repository
2. Run the unit tests:
   ```bash
   go test -v ./internal/patterns/learning/...
   ```
3. Run the integration tests:
   ```bash
   go test -v ./internal/patterns/learning/integration/...
   ```
4. Set up the Docker test environment:
   ```bash
   docker build -t sortd-test -f Dockerfile.test .
   ```
5. Run the Docker test scenarios