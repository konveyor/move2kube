import yaml


def filter_obj(x):
    return (
        x["label"],
        {
            "label": x["label"],
            "description": x["description"],
            "servicebrokername": x["servicebrokername"],
            "tags": x["tags"],
            # "extra": x["extra"],
        },
    )


def process_cf_services(input_path, output_path):
    with open(input_path) as f:
        cf_services = yaml.safe_load(f)
    # print('cf_services', cf_services)
    # print("cf_services", type(cf_services))
    # print(cf_services.keys())
    # print(cf_services["spec"].keys())
    # print(type(cf_services["spec"]["services"]))
    # print(len(cf_services["spec"]["services"]))
    catalogue = {}
    for x in cf_services["spec"]["services"]:
        # print(type(x))
        # print(x.keys())
        k, v = filter_obj(x)
        # print(v)
        catalogue[k] = v
        # print("-" * 100)
    with open(output_path, "w") as f:
        yaml.dump(catalogue, f)

if __name__ == "__main__":
    process_cf_services("cfservices.yaml", "catalogue.yaml")
