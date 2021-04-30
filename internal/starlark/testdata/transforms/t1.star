"""some transforms for migrating myapp"""

def select_gpu_nodes(x):
    x["metadata"]["annotations"]["openshift.io/node-selector"] = "type=gpu-node,region=west"
    return x

def lower_number_of_replicas(x):
    x["spec"]["replicas"] = 2
    return x

def change_the_image(x):
    if x["metadata"]["name"] == "javaspringapp":
        x["spec"]["template"]["spec"]["containers"][0]["image"] = query({"key":"services.javaspringapp.containers.[0].image"})
    return x

outputs = {
    "transforms": [
        {"transform": "select_gpu_nodes", "filter": {"Namespace": ["v1"]}},
        {"transform": "lower_number_of_replicas", "filter": {"Deployment": ["apps/v1", "extensions/v1beta1"]}},
        {"transform": "change_the_image", "filter": {"Deployment": ["^.*/v1.*$"]}},
    ],
}
