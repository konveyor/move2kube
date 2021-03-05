"""some transforms for migrating myapp"""

def select_gpu_nodes(x):
    x["metadata"]["annotations"]["openshift.io/node-selector"] = "type=gpu-node,region=west"
    return x

def lower_number_of_replicas(x):
    x["spec"]["replicas"] = 2
    return x

def change_the_ports(x):
    x["spec"]["template"]["spec"]["containers"][0]["image"] = answers("services.svc1.containers.[0].image")
    return x

outputs = {
    "questions": [
        {
            "key": "services.svc1.containers.[0].image",
            "description": "What image should svc1 use?",
        },
    ],
    "transforms": [
        {"transform": "select_gpu_nodes", "filter": {"Namespace": ["v1"]}},
        {"transform": "lower_number_of_replicas", "filter": {"Deployment": ["apps/v1", "extensions/v1beta1"]}},
        {"transform": "change_the_ports", "filter": {"Deployment": ["^.*/v1.*$"]}},
    ],
}
