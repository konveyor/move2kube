/*
 *  Copyright IBM Corporation 2022
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

package graph

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	graphtypes "github.com/konveyor/move2kube/types/graph"
	"github.com/sirupsen/logrus"
)

// content is our static web server content.
//go:embed web/build/*
var content embed.FS

// StartServer starts the graph server and web UI to display the nodes and edges.
func StartServer(nodes []graphtypes.Node, edges []graphtypes.EdgeT, port int32) error {
	graph := map[string]interface{}{"nodes": nodes, "edges": edges}
	jsonBytes, err := json.Marshal(graph)
	if err != nil {
		logrus.Fatalf("failed to marshal the graph to json. Error: %q", err)
	}
	router := mux.NewRouter()
	sub, _ := fs.Sub(content, "web/build")
	router.Path("/graph.json").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(jsonBytes); err != nil {
			logrus.Errorf("failed to write the json bytes out to the response. Actual:\n%s\nError: %q", string(jsonBytes), err)
		}
	})
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.FS(sub))))
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	logrus.Infof("Listening on http://%s/", addr)
	return server.ListenAndServe()
}
