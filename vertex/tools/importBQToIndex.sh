curl -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json; charset=utf-8" \
  -d "@vertex/tools/importBigIndex.json" \
  "https://$LOCATION-aiplatform.googleapis.com/v1beta1/projects/$GCP_PROJECT_ID/locations/$LOCATION/indexes/$VERTEX_INDEX_ID:import"
   