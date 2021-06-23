# Image Processor

steps:

- define input images: fill the `data/input_images`
- login with skopeo in the terminal: `skopeo login docker.io`
- run image crawler: `sh run_image_crawler.sh`
  - this will generate a json file per image in `output/` and also a json file per tag in `output/image`  
 - run the aggregator`python run_aggreagator.py`
   - extract java version from image, parse it. 
   - save aggregated result as `output/consolidated_images.json`