/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package questionreceivers

//import (
//	"context"
//	"fmt"
//	"net"
//
//	"github.com/konveyor/move2kube-wasm/qaengine"
//	qatypes "github.com/konveyor/move2kube-wasm/types/qaengine"
//	qagrpc "github.com/konveyor/move2kube-wasm/types/qaengine/qagrpc"
//	"github.com/phayes/freeport"
//	"github.com/sirupsen/logrus"
//	"github.com/spf13/cast"
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/reflection"
//)
//
//var (
//	grpcReceiver net.Addr
//)
//
//type server struct {
//	qagrpc.UnimplementedQAEngineServer
//}
//
//func (s *server) FetchAnswer(ctx context.Context, prob *qagrpc.Problem) (a *qagrpc.Answer, err error) {
//	logrus.Debugf("Received Question over grpc : %+v", prob)
//	qaprob, err := qatypes.NewProblem(prob)
//	if err != nil {
//		logrus.Errorf("Unable to read problem : %s", err)
//		return a, err
//	}
//	qaans, err := qaengine.FetchAnswer(qaprob)
//	if err != nil {
//		logrus.Errorf("Unable to get answer : %s", err)
//		return a, err
//	}
//	a = &qagrpc.Answer{}
//	a.Answer, err = qatypes.InterfaceToArray(qaans.Answer, qaans.Type)
//	if err != nil {
//		logrus.Errorf("Unable to interpret answer : %s", err)
//	}
//	return a, err
//}
//
//// StartGRPCReceiver starts the GRPC receiver for QA Engine
//func StartGRPCReceiver() (addr net.Addr, err error) {
//	if grpcReceiver != nil {
//		return grpcReceiver, nil
//	}
//	port, err := freeport.GetFreePort()
//	if err != nil {
//		return addr, fmt.Errorf("unable to find a free port : %s", err)
//	}
//	portstr := cast.ToString(port)
//	listener, err := net.Listen("tcp", ":"+portstr)
//	if err != nil {
//		logrus.Errorf("failed to listen: %v", err)
//		return addr, err
//	}
//	s := grpc.NewServer()
//	qagrpc.RegisterQAEngineServer(s, &server{})
//	reflection.Register(s)
//	logrus.Debugf("server for grpc QA Receiver listening at %v", listener.Addr())
//	go func(listener net.Listener) {
//		err := s.Serve(listener)
//		if err != nil {
//			logrus.Fatalf("Unable to start qa receiver engine : %s", err)
//		}
//	}(listener)
//	logrus.Info("Started QA GPRC Receiver engine on: " + listener.Addr().String())
//	grpcReceiver = listener.Addr()
//	return grpcReceiver, nil
//}
