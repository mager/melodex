# melodex

## Development

Call Firestore locally:

```sh
gcloud iam service-accounts create local-dev
gcloud projects add-iam-policy-binding beatbrain-dev --member="serviceAccount:local-dev@beatbrain-dev.iam.gserviceaccount.com" --role="roles/owner"
gcloud iam service-accounts keys create credentials.json --iam-account=local-dev@beatbrain-dev.iam.gserviceaccount.com
```

Set credentials:

```sh
export GOOGLE_APPLICATION_CREDENTIALS=$(echo $(pwd)/credentials.json)
```
