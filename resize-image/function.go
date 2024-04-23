package function

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/disintegration/imaging"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/objectstorage"
	"image"
	"io"
	"log"
	"os"
	"sync"
)

var client *objectstorage.ObjectStorageClient
var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

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
	Message *PubSubMessage `json:"message"`
}

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Data *Payload `json:"data"`
}

type Payload struct {
	ObjectName   string `json:"objectName"`
	TargetName   string `json:"targetName"`
	TargetWidth  int    `json:"targetWidth"`
	TargetHeight int    `json:"targetHeight"`
}

// resizeImage resizes the input image and returns the resized image
func resizeImage(img image.Image, width int, height int) image.Image {
	// Implement image resizing logic using the disintegration/imaging package
	return imaging.Resize(img, width, height, imaging.Box)
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

	// Read the image from OCI Object Storage as a stream
	imageStream, err := readFromOCIObjectStorage(ctx, namespace, bucket, msg.Message.Data.ObjectName)
	if err != nil {
		return fmt.Errorf("error reading image from OCI Object Storage: %v", err)
	}
	defer func(imageStream io.ReadCloser) {
		_ = imageStream.Close()
	}(imageStream)

	// Decode the image
	img, err := imaging.Decode(imageStream)
	if err != nil {
		return fmt.Errorf("error decoding image: %v", err)
	}

	// Resize the image
	resizedImg := resizeImage(img, msg.Message.Data.TargetWidth, msg.Message.Data.TargetHeight)

	// Save the resized image to OCI Object storage
	if err := saveToOCIObjectStorage(ctx, &resizedImg, namespace, bucket, msg.Message.Data.TargetName); err != nil {
		return fmt.Errorf("error saving resized image to OCI Object Storage: %v", err)
	}

	return nil
}

// readFromOCIObjectStorage reads the image from OCI Object Storage as a stream
func readFromOCIObjectStorage(ctx context.Context, namespaceName, bucketName, objectName string) (io.ReadCloser, error) {
	getObjectRequest := objectstorage.GetObjectRequest{
		NamespaceName: &namespaceName,
		BucketName:    &bucketName,
		ObjectName:    &objectName,
	}

	getObjectResponse, err := client.GetObject(ctx, getObjectRequest)
	if err != nil {
		return nil, err
	}

	return getObjectResponse.HTTPResponse().Body, nil
}

// saveToOCIObjectStorage saves the image to OCI Object Storage as a stream
func saveToOCIObjectStorage(ctx context.Context, img *image.Image, namespaceName, bucketName, objectName string) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer func() {
		// 너무 큰 버퍼는 해제
		if buf.Cap() > 10*1024*1024 {
			return
		}
		buf.Reset()
		bufPool.Put(buf)
	}()

	if err := imaging.Encode(buf, *img, imaging.JPEG, imaging.JPEGQuality(80)); err != nil {
		return err
	}
	// upload image to OCI Object Storage
	imageReader := io.NopCloser(bytes.NewReader(buf.Bytes()))
	contentLength := int64(buf.Len())
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
