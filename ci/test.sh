#!/usr/bin/env bash

set -eou pipefail

hash() {
  if command -v md5sum >/dev/null 2>&1; then
    md5sum "$@"
  else
    md5 "$@"
  fi
}

event() {
  local source_uri="${1}"
  local destination_uri="${2}"
  local file_upload_uri="${3}"
  local mimetype="${4}"
  local args=""
  if [ "$#" -eq 5 ]; then
    args="${5}"
  fi
  cat << EOT
{
  "actor": {
    "id": "urn:uuid:01abcdef-2345-6789-abcd-ef0123456789"
  },
  "object": {
    "id": "urn:uuid:abcdef01-2345-6789-abcd-ef0123456789",
    "url": [
      {
        "name": "Canonical",
        "type": "Link",
        "href": "https://islandora.dev/node/1",
        "mediaType": "text/html",
        "rel": "canonical"
      },
      {
        "name": "JSON",
        "type": "Link",
        "href": "https://islandora.dev/node/1?_format=json",
        "mediaType": "application/json",
        "rel": "alternate"
      },
      {
        "name": "JSONLD",
        "type": "Link",
        "href": "https://islandora.dev/node/1?_format=jsonld",
        "mediaType": "application/ld+json",
        "rel": "alternate"
      }
    ],
    "isNewVersion": true
  },
  "attachment": {
    "type": "Object",
    "content": {
      "mimetype": "$mimetype",
      "args": "$args",
      "source_uri": "$source_uri",
      "destination_uri": "$destination_uri",
      "file_upload_uri": "$file_upload_uri"
    },
    "mediaType": "application/json"
  },
  "type": "Activity",
  "summary": "Generate Derivative"
}
EOT
}

apk update && apk add jq

echo "Triggering tests"

SERVICES=(
  "crayfits"
  "homarus"
  "pandoc"
)
for SERVICE in "${SERVICES[@]}"; do
  echo "Testing $SERVICE"

  if [ "$SERVICE" == "crayfits" ]; then
    source_uri="http://file-server/files/crayfits/test.txt"
    destination_uri="http://file-server/"
    file_upload_uri="/tmp/result.xml"
    mimetype="application/xml"
    args=""
    curl -s -u admin:password \
      -X POST \
      -o /dev/null \
      -H "Content-Type: application/json" \
      -d "$(event $source_uri $destination_uri $file_upload_uri $mimetype $args)" \
      "http://activemq:8161/api/message/islandora-connector-fits?type=queue"

    while [ ! -f "$file_upload_uri" ]; do
      echo "waiting for $file_upload_uri to get created"
      sleep 1
    done

    echo "Making sure the fits XML has the correct checksum"
    grep acbd18db4cc2f85cedef654fccc4a4d8 "$file_upload_uri" | grep md5checksum && echo "FITS ran successfully"
    rm $file_upload_uri

  elif [ "$SERVICE" == "homarus" ]; then
    source_uri="http://file-server/files/homarus/bunny.mp4"
    destination_uri="http://file-server/"
    file_upload_uri="/tmp/image.jpg"
    mimetype="image/jpeg"
    args="-ss 00:00:45.000 -frames 1 -vf scale=720:-2"
    curl -s -u admin:password \
      -X POST \
      -o /dev/null \
      -H "Content-Type: application/json" \
      -d "$(event $source_uri $destination_uri $file_upload_uri $mimetype \"$args\")" \
      "http://activemq:8161/api/message/islandora-connector-$SERVICE?type=queue"

    while [ ! -f "$file_upload_uri" ]; do
      echo "waiting for $file_upload_uri to get created"
      sleep 1
    done

    hash "$file_upload_uri" | grep fe7dd57460dbaf50faa38affde54b694 && echo "homarus/ffmpeg ran successfully"
    rm "$file_upload_uri"

  elif [ "$SERVICE" == "pandoc" ]; then
    source_uri="http://file-server/files/pandoc/input.md"
    destination_uri="http://file-server/"
    file_upload_uri="/tmp/result.tex"
    mimetype="application/x-latex"
    args=""
    curl -s -u admin:password \
      -X POST \
      -o /dev/null \
      -H "Content-Type: application/json" \
      -d "$(event $source_uri $destination_uri $file_upload_uri $mimetype $args)" \
      "http://activemq:8161/api/message/islandora-connector-$SERVICE?type=queue"

    while [ ! -f "$file_upload_uri" ]; do
      echo "waiting for $file_upload_uri to get created"
      sleep 1
    done

    if diff -u $file_upload_uri /tmp/pandoc/output.tex > diff_output.txt; then
      echo "Test Passed: Output matches expected."
    else
      echo "Test Failed: Differences found."
      cat diff_output.txt
      exit 1
    fi
    rm $file_upload_uri
  else
    echo "Unknown service"
    exit 1
  fi
done
