#!/bin/bash

# image names are defined in a plain file
input="data/input_images.txt"
while IFS= read -r image
do
  echo $image
  echo "============="
  savefile="${image////_sep_}"
  data=$(skopeo inspect --override-os linux "docker://docker.io/$image")
  
  #save file
  echo $data > "output/$savefile".json

  # now we crawl each tag

  # create a folder to store each json
  mkdir -p output/$savefile

  # collect the tags
  tags=$(echo $data | jq -r ".RepoTags[]")
  
  echo "crawling tags"
  echo "-------------"
  for tag in $tags[@]; do
    echo " "$tag

    # we dont crawl again (for the moment)
    if [ -f "/Users/pablo/Desktop/image_processor_output/$savefile/$tag.json" ]; then
      echo "  file exists"
    else
      echo "  file does not exist -> crawling"
      skopeo inspect --config --override-os linux "docker://docker.io/$image:$tag" > "/Users/pablo/Desktop/image_processor_output/$savefile/$tag".json
      sleep 5
      # https://docs.docker.com/docker-hub/download-rate-limit/
    fi  

  done

done < "$input"