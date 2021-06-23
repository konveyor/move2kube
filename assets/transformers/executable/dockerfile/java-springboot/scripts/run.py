#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

import ipdb
import argparse

from framework_detector import FrameworkDetector
from framework_parameter_extractor import (FrameworkParameterExtractor, 
                                        FrameworkParameterExtractorHeuristic)
from server_detector import ServerDetector
from server_parameter_extractor import (ServerParameterExtractor, 
                                        ServerParameterExtractorFromImage)

from pcf_extractor import PCFExtractor
from segment_integrator import SegmentIntegrator
from metadata_extractor import MetadataExtractor
from data_loader import DataLoader


def main(args):

    # ------------------
    # Load captured data
    # ------------------
    dl = DataLoader(args.output_path)
    captured_data = dl.load_captured_data()

    # ------------------
    # Metadata extractor
    # ------------------
    md = MetadataExtractor(captured_data, args.app_path, args.basename)
    metadata = md.get_all_metadata()
    
    
    # ------------------
    # Framework detector
    # ------------------
    fw = FrameworkDetector(captured_data, args.app_path)
    fw_result = fw.detect()

    # -----------------------------
    # Framework Parameter extractor
    # -----------------------------
    framework_params = {}
    if any([v == True for k,v in fw_result.items()]):
        # At least one framework was detected
        for framework, result in fw_result.items():
            if result == True:
                fpe = FrameworkParameterExtractor(captured_data, args.basename)
                framework_params[framework] = fpe.extract(framework)

    else:
        # No framework was detected, we need to discover 
        # parameters heuristically.
        fpeh = FrameworkParameterExtractorHeuristic(captured_data)
        framework_params["undefined"] = fpeh.extract()

    # ---------------------------
    # Application server detector
    # ---------------------------
    sd = ServerDetector(captured_data, framework_params)
    server_detection_result = sd.detect()
    
    
    # --------------------------------------
    # Application server parameter extractor
    # --------------------------------------
    spe = ServerParameterExtractor(captured_data, 
                                    server_detection_result,
                                    metadata)
    server_params = spe.extract()


    # -------------------------------------
    # CloudFoundry parameter extractor
    # -------------------------------------
    pcf = PCFExtractor(captured_data)
    pcf_params = pcf.extract()
    
    # ---------------------
    # Segments Integrator
    # ---------------------
    si = SegmentIntegrator(
            framework_params,
            server_params,
            metadata, 
            pcf_params, 
            build_type="maven")
    
    si.construct_segment_list()
    json_output = si.get_json_output()

    si.persist_full_template()

    # Print result as a json output
    # -----------------------------
    print(json_output)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--app_path", type=str, 
        default="Absolute path of the original application")
    parser.add_argument("--output_path", type=str, 
        default="Location of the files generate by the bash script")
    parser.add_argument("--basename", type=str, 
        default="Basename of the project")
    parser.add_argument("--input_type", type=str, 
        default="Can be `directory or `file`")

    args = parser.parse_args()
    main(args)