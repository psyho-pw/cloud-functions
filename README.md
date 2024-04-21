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
curl --location 'localhost:8080/projects/sayho-general/topics/test' \
--header 'ce-id: 123451234512345' \
--header 'ce-specversion:  1.0' \
--header 'ce-time:  2020-01-02T12:34:56.789Z' \
--header 'ce-type:  google.cloud.pubsub.topic.v1.messagePublished' \
--header 'ce-source:  //pubsub.googleapis.com/projects/MY-PROJECT/topics/MY-TOPIC' \
--header 'Content-Type: application/json' \
--data '{
    "message": {
        "data": {
            "objectName": "test.jpg"
        }
    }
}'
```