import os
import grpc
import fetchanswer_pb2_grpc as pb2_grpc
import fetchanswer_pb2 as pb2
import ipdb

class Client():
    def __init__(self):
        #self.host = "localhost"
        #self.server_port = 50051

        address = os.getenv("GRPC_RECEIVER")
        self.channel = grpc.insecure_channel(address)
        #    '{}:{}'.format(self.host, self.server_port))

        self.stub = pb2_grpc.QAEngineStub(self.channel)

    def get_answer(self, id, type, hints, options, default):

        problem = pb2.Problem(id = id,
        type= type,
        hints = hints,
        options = options,
        default= default)

        #print(f'{problem}')
        return self.stub.FetchAnswer(problem)

if __name__ == "__main__":
    
    client = Client()
    result = client.get_answer("move2kube.xyz","MultiSelect",[], ["a","b"], [])
    print(f'{result}')