# Ingestion Sequence

* Participants
* Expected sequence
* Endpoints
* Data

```mermaid
sequenceDiagram
  participant S as Scicat Backend
  participant U as Ingestor UI 
  participant B as Ingestor Backend 
  participant M as Metadata Extractor

    S -->> U: Serve Ingestor UI
    U -->> B: Establish Connection: GET /version
    activate B
    B -->> U: version
    deactivate B

    U -->> B: Get Available Extractors: GET /extractors
    activate B
    B -->> U: extractor names
    deactivate B

    U -->> B: Get Folder List: GET /datasets
    activate B
    B -->> U: folders
    deactivate B
    
    activate B
    U -->> B: Extract Metadata: POST /extract {folder}
    B -->> M: Invoke Extractor
    activate M
    M -->> M: write metadata.json
    M -->> B: return status code
    deactivate M
    B -->> B: read metadata.json
    B -->> U: return metdata.json
    deactivate B

    U -->> U: Display extracted metadata

    U -->> U: Add user metadata

    U -->> B: Start ingestion: POST /transfer {folder, metadata}
    
```
