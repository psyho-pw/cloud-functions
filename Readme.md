# GCP Cloud Functions collection

### Local test 

#### install functions framework library
```bash
go install github.com/GoogleCloudPlatform/functions-framework-go/funcframework
```

#### move to target function directory
```cd resize-image```
#### run command & test
```bash 
make start
````
#### make HTTP POST call
```bash
curl --location 'localhost:8080' \
--header 'Content-Type: application/json' \
--data '{
    
}'
```