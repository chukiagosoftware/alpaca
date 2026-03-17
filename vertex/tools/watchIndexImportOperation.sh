OPERATION="projects/$GCP_PROJECT_ID/locations/$LOCATION/operations/$1"
while true; do
  clear
  date
  gcloud ai operations describe $OPERATION --project=$GCP_PROJECT_ID
  sleep 60 # Wait for 5 seconds before checking again
done