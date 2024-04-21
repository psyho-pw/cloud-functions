package function

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/nfnt/resize"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/objectstorage"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
)

var client *objectstorage.ObjectStorageClient

func createObjectStorageClient() (*objectstorage.ObjectStorageClient, error) {
	tenancy := os.Getenv("OCI_TENANCY")
	user := os.Getenv("OCI_USER")
	region := os.Getenv("OCI_REGION")
	fingerprint := os.Getenv("OCI_FINGERPRINT")
	privateKeyEncoded := os.Getenv("OCI_PRIVATE_KEY")
	privateKey, _ := base64.StdEncoding.DecodeString(privateKeyEncoded)
	passPhrase := os.Getenv("OCI_PASSPHRASE")

	provider := common.NewRawConfigurationProvider(
		tenancy,
		user,
		region,
		fingerprint,
		string(privateKey),
		&passPhrase,
	)

	//provider := common.ConfigurationProviderEnvironmentVariables("OCI", passPhrase)

	objectStorageClient, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
	if err != nil {
		log.Printf("%+v", err)
		return nil, err
	}

	return &objectStorageClient, nil
}

func init() {
	var clientInitErr error
	client, clientInitErr = createObjectStorageClient()
	if clientInitErr != nil {
		panic("client init failure")
	}

	functions.CloudEvent("ResizeImage", handlePubSubMessage)
}

// MessagePublishedData contains the full Pub/Sub message
// See the documentation for more details:
// https://cloud.google.com/eventarc/docs/cloudevents#pubsub
type MessagePublishedData struct {
	Message PubSubMessage `json:"message"`
}

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Data Payload `json:"data"`
}

type Payload struct {
	ObjectName string `json:"objectName"`
}

// resizeImage resizes the input image and returns the resized image
func resizeImage(img image.Image) image.Image {
	// Implement image resizing logic using the nfnt/resize package
	resizedImg := resize.Resize(200, 0, img, resize.Lanczos3)
	return resizedImg
}

// resizeImage consumes a CloudEvent message and extracts the Pub/Sub message.
func handlePubSubMessage(ctx context.Context, e event.Event) error {
	var msg MessagePublishedData
	if err := e.DataAs(&msg); err != nil {
		return fmt.Errorf("event.DataAs: %v", err)
	}

	// target info
	namespace := os.Getenv("OCI_NAMESPACE")
	bucket := os.Getenv("OCI_BUCKET")

	// Decode base64 encoded image data
	imageData, err := readFromOCIObjectStorage(ctx, namespace, bucket, msg.Message.Data.ObjectName)
	if err != nil {
		return fmt.Errorf("error reading image from OCI Object Storage: %v", err)
	}

	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	// Resize the image
	resizedImg := resizeImage(img)

	// Save the resized image to OCI Object storage
	if err := saveToOCIObjectStorage(ctx, resizedImg, namespace, bucket, "resized-image.jpg"); err != nil {
		return fmt.Errorf("error saving resized image to OCI Object Storage: %v", err)
	}

	return nil
}

func readFromOCIObjectStorage(ctx context.Context, namespaceName, bucketName, objectName string) ([]byte, error) {
	getObjectRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespaceName,
		BucketName:    &bucketName,
		ObjectName:    &objectName,
	}

	getObjectResponse, err := client.GetObject(ctx, getObjectRequest)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("%+v", err)
		}
	}(getObjectResponse.HTTPResponse().Body)

	var buf bytes.Buffer
	_, err = io.Copy(&buf, getObjectResponse.HTTPResponse().Body)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func saveToOCIObjectStorage(ctx context.Context, img image.Image, namespaceName, bucketName, objectName string) error {
	// 이미지 데이터를 바이트 슬라이스로 인코딩
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		return err
	}
	imageData := buf.Bytes()

	// bytes.NewReader(imageData)를 io.ReadCloser로 변환
	imageReader := io.NopCloser(bytes.NewReader(imageData))
	// OCI Object Storage에 이미지 업로드
	contentLength := int64(len(imageData))
	putObjectRequest := objectstorage.PutObjectRequest{
		NamespaceName: &namespaceName,
		BucketName:    &bucketName,
		ObjectName:    &objectName,
		PutObjectBody: imageReader,
		ContentLength: &contentLength,
	}

	_, err := client.PutObject(ctx, putObjectRequest)
	if err != nil {
		return err
	}

	log.Printf("Resized image saved to OCI Object Storage")
	return nil
}
