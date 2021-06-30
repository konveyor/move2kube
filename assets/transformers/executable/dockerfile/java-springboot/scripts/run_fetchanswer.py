import os
import grpc
import fetchanswer_pb2_grpc as pb2_grpc
import fetchanswer_pb2 as pb2
import ipdb

class GRPCClient():
    def __init__(self):
        # get server address from env variable
        address = os.getenv("GRPC_RECEIVER")

        self.channel = grpc.insecure_channel(address)
        self.stub = pb2_grpc.QAEngineStub(self.channel)

    def get_answer(self, id, type, hints, options, default):

        problem = pb2.Problem(
            id = id,
            type= type,
            hints = hints,
            options = options,
            default= default
        )

        return self.stub.FetchAnswer(problem)

if __name__ == "__main__":
    
    client = GRPCClient()
    result = client.get_answer(
        "move2kube.xyz", # id
        "MultiSelect",   # type
        [],              # hints
        ["a","b"],       # options
        []               # default
    )