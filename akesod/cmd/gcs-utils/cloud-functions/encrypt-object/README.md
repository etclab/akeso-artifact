- Ensure `roles/pubsub.publisher` [role is assigned to the service agent](https://cloud.google.com/storage/docs/reporting-changes#grant-required-role-to-service-agent) in your project. Assign the role using: 
  ```bash
  export PROJECT_ID="wild-flame-12345"
  export PROJECT_NUMBER=$(gcloud projects describe $PROJECT_ID --format="value(projectNumber)" | tail -1)
  gcloud projects add-iam-policy-binding $PROJECT_ID --member=serviceAccount:service-${PROJECT_NUMBER}@gs-project-accounts.iam.gserviceaccount.com --role=roles/pubsub.publisher
  ```

- First setup a notification config in cloud storage bucket. This sends the object's metadata update event (pub/sub message) to the given topic id. The key for encrypting object is attached within the event's custom attribute.

  Example:

  ```bash
  ./gcs-utils -notification-config -topic-id MetadataUpdate -project-id $PROJECT_ID -event-type OBJECT_METADATA_UPDATE -custom-attributes=new_dek=x++yudvAtO9scc4NPYQusmPD0bsiyQic9ZHziqu62DE= gs://<BUCKET>
  ```

- Next setup a cloud function that receives the event (pub/sub message) and encrypts the object.

  Example (inside the cloud-functions/encrypt-object dir)

  ```bash
  gcloud functions deploy encrypt-object \
    --gen2 \
    --runtime=go122 \
    --region=us-east1 \
    --source=. \
    --entry-point=EncryptObject \
    --trigger-topic=MetadataUpdate
  ```
