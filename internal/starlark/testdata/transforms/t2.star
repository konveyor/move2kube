"""some more transforms for migrating myapp"""

def change_the_container_name(x):
    new_name = ask_ques({
        "key": 'services."{}".containers.[0].name'.format(x["metadata"]["name"]),
        "description": "What should be the new name for the container {} ?".format(x["spec"]["template"]["spec"]["containers"][0]["name"]),
    })
    x["spec"]["template"]["spec"]["containers"][0]["name"] = new_name
    return x

outputs = {
    "transforms": [
        {"transform": "change_the_container_name", "filter": {"Deployment": ["^.*/v1.*$"]}},
    ],
}
