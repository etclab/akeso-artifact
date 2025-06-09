package gcsx

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strings"

	"cloud.google.com/go/storage"
)

// Returns bucketName, objectName, error
func ParseUrl(urlStr string) (string, string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", err
	}

	if u.Scheme != "gs" {
		err := fmt.Errorf("gcsx.ParseUrl: bad url %q: expected scheme of \"gs\" but got %q", urlStr, u.Scheme)
		return "", "", err
	}

	bucketName := u.Host
	objectName := u.Path

	if strings.HasPrefix(objectName, "/") {
		objectName, _ = strings.CutPrefix(objectName, "/")
	}

	return bucketName, objectName, nil
}

func GetObject(obj *storage.ObjectHandle) ([]byte, error) {
	ctx := context.Background()
	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func PutObject(obj *storage.ObjectHandle, data []byte) error {
	ctx := context.Background()
	w := obj.NewWriter(ctx)
	_, err := w.Write(data)
	if err != nil {
		return err
	}
	return w.Close()
}

func UpdateObjectMetadata(obj *storage.ObjectHandle, metadata map[string]string) error {
	attrsToUpdate := storage.ObjectAttrsToUpdate{
		Metadata: metadata,
	}
	ctx := context.Background()
	_, err := obj.Update(ctx, attrsToUpdate)
	if err != nil {
		return err
	}
	return nil
}

func PutObjectWithMetadata(obj *storage.ObjectHandle, data []byte, metadata map[string]string) error {
	ctx := context.Background()
	w := obj.NewWriter(ctx)
	w.Metadata = metadata
	_, err := w.Write(data)
	if err != nil {
		fmt.Printf("Error writing object: %v\n", err)
		return err
	}

	return w.Close()
}

func AddNotification(ctx context.Context, bucket *storage.BucketHandle,
	notification *storage.Notification) (*storage.Notification, error) {
	notif, err := bucket.AddNotification(ctx, notification)
	if err != nil {
		return nil, err
	}
	return notif, nil
}

// removes notification configs matching the topicId, projectId, and eventType
func RemoveNotification(ctx context.Context, bucket *storage.BucketHandle,
	topicId string, projectId string, eventType string) error {

	notifList, err := bucket.Notifications(ctx)
	if err != nil {
		return err
	}

	for id, notification := range notifList {
		if topicId == notification.TopicID &&
			projectId == notification.TopicProjectID &&
			slices.Contains(notification.EventTypes, eventType) {

			err = RemoveNotificationById(ctx, bucket, id)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// remove a notification config from bucket
func RemoveNotificationById(ctx context.Context, bucket *storage.BucketHandle,
	notificationId string) error {
	err := bucket.DeleteNotification(ctx, notificationId)
	if err != nil {
		return err
	}
	return nil
}

func DumpObjectAttrs(w io.Writer, attrs *storage.ObjectAttrs) {
	fmt.Fprintf(w, "Bucket: %s\n", attrs.Bucket)
	fmt.Fprintf(w, "Name: %s\n", attrs.Name)
	fmt.Fprintf(w, "ContentType: %s\n", attrs.ContentType)
	fmt.Fprintf(w, "ContentLangauge: %s\n", attrs.ContentLanguage)
	fmt.Fprintf(w, "CacheControl: %s\n", attrs.CacheControl)
	fmt.Fprintf(w, "EventBasedHold: %t\n", attrs.EventBasedHold)
	fmt.Fprintf(w, "TemporaryHold: %t\n", attrs.TemporaryHold)
	fmt.Fprintf(w, "RetentionExpirationTime: %v\n", attrs.RetentionExpirationTime)
	fmt.Fprintf(w, "ACL: %v\n", attrs.ACL)
	fmt.Fprintf(w, "PredefinedACL: %v\n", attrs.PredefinedACL)
	fmt.Fprintf(w, "Owner: %s\n", attrs.Owner)
	fmt.Fprintf(w, "Size: %d\n", attrs.Size)
	fmt.Fprintf(w, "ContentEncoding: %s\n", attrs.ContentEncoding)
	fmt.Fprintf(w, "MD5: %s\n", hex.EncodeToString(attrs.MD5))
	fmt.Fprintf(w, "CRC32C: %x\n", attrs.CRC32C)
	fmt.Fprintf(w, "MediaLink: %s\n", attrs.MediaLink)
	fmt.Fprintf(w, "Generation: %d\n", attrs.Generation)
	fmt.Fprintf(w, "Metageneration: %d\n", attrs.Metageneration)
	fmt.Fprintf(w, "StorageClass: %s\n", attrs.StorageClass)
	fmt.Fprintf(w, "Created: %v\n", attrs.Created)
	fmt.Fprintf(w, "Deleted: %v\n", attrs.Deleted)
	fmt.Fprintf(w, "Updated: %v\n", attrs.Updated)
	fmt.Fprintf(w, "CustomerKeySHA256: %s\n", attrs.CustomerKeySHA256)
	fmt.Fprintf(w, "KMSKeyName: %s\n", attrs.KMSKeyName)
	fmt.Fprintf(w, "Prefix: %s\n", attrs.Prefix)
	fmt.Fprintf(w, "Etag: %s\n", attrs.Etag)
	fmt.Fprintf(w, "ComponentCount: %d\n", attrs.ComponentCount)

	fmt.Fprintf(w, "Metadata: {\n")
	for k, v := range attrs.Metadata {
		fmt.Fprintf(w, "    %s: %s\n", k, v)
	}
	fmt.Fprintf(w, "}\n")
}
