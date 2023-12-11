/*
 * Tencent is pleased to support the open source community by making TKEStack available.
 *
 * Copyright (C) 2012-2019 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */
package route

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/julienschmidt/httprouter"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	extenderv1 "k8s.io/kube-scheduler/extender/v1"
	"tkestack.io/gpu-admission/pkg/predicate"
	"tkestack.io/gpu-admission/pkg/version"
	vapi "volcano.sh/volcano/pkg/scheduler/api"
	vextender "volcano.sh/volcano/pkg/scheduler/plugins/extender"
)

const (
	// version router path
	versionPath = "/version"
	apiPrefix   = "/scheduler"
	// predication router path
	predicatesPrefix  = apiPrefix + "/predicates"
	predicatesVPrefix = apiPrefix + "/vpredicates"
)

func checkBody(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
}

// PredicateRoute sets router table for predication
func PredicateRoute(predicate predicate.Predicate) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		checkBody(w, r)

		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)

		var extenderArgs extenderv1.ExtenderArgs
		var extenderFilterResult *extenderv1.ExtenderFilterResult

		if err := json.NewDecoder(body).Decode(&extenderArgs); err != nil {
			extenderFilterResult = &extenderv1.ExtenderFilterResult{
				Nodes:       nil,
				FailedNodes: nil,
				Error:       err.Error(),
			}
		} else {
			extenderFilterResult = predicate.Filter(extenderArgs)
			klog.V(4).Infof("%s: ExtenderArgs = %+v", predicate.Name(), extenderArgs)
		}

		if resultBody, err := json.Marshal(extenderFilterResult); err != nil {
			klog.Errorf("Failed to marshal extenderFilterResult: %+v, %+v",
				err, extenderFilterResult)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			klog.V(4).Infof("%s: extenderFilterResult = %s",
				predicate.Name(), string(resultBody))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(resultBody)
		}
	}
}

func VPredicateRoute(predicate predicate.Predicate) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		checkBody(w, r)

		var buf bytes.Buffer
		body := io.TeeReader(r.Body, &buf)
		req := &vextender.PredicateRequest{}
		resp := &vextender.PredicateResponse{}

		var extenderArgs extenderv1.ExtenderArgs
		var extenderFilterResult *extenderv1.ExtenderFilterResult

		if err := json.NewDecoder(body).Decode(&req); err != nil {
			// error
		} else {
			extenderArgs.Nodes = &v1.NodeList{Items: []v1.Node{*(req.Node.Node)}}
			extenderArgs.Pod = req.Task.Pod
			extenderFilterResult = predicate.Filter(extenderArgs)
			klog.V(4).Infof("%s: ExtenderArgs = %+v", predicate.Name(), extenderArgs)
		}
		if extenderFilterResult.Error != "" {
			vstatus := &vapi.Status{Reason: extenderFilterResult.Error, Code: vapi.Error}
			resp.Status = append(resp.Status, vstatus)
		} else {
			vstatus := &vapi.Status{Reason: "success", Code: vapi.Success}
			resp.Status = append(resp.Status, vstatus)
		}

		if resultBody, err := json.Marshal(resp); err != nil {
			klog.Errorf("Failed to marshal resp: %+v, %+v",
				err, resp)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			klog.V(4).Infof("%s: extenderFilterResult = %s",
				predicate.Name(), string(resultBody))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(resultBody)
		}
	}
}

// VersionRoute returns the version of router in response
func VersionRoute(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, fmt.Sprint(version.Get()))
}

func AddVersion(router *httprouter.Router) {
	router.GET(versionPath, DebugLogging(VersionRoute, versionPath))
}

// DebugLogging wraps handler for debugging purposes
func DebugLogging(h httprouter.Handle, path string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		klog.Infof("enter predicates")
		klog.V(10).Infof("%s request body = %s", path, r.Body)
		h(w, r, p)
		klog.Infof("after predicates")
		klog.V(10).Infof("%s response = %s", path, w)
	}
}

func AddPredicate(router *httprouter.Router, predicate predicate.Predicate) {
	path := predicatesPrefix
	klog.Infof("register %s", path)
	router.POST(path, DebugLogging(PredicateRoute(predicate), path))
}

func AddVPredicate(router *httprouter.Router, predicate predicate.Predicate) {
	path := predicatesVPrefix
	klog.Infof("register %s", path)
	router.POST(path, DebugLogging(VPredicateRoute(predicate), path))
}

/*
- name: extender
        arguments:
          extender.urlPrefix: http://127.0.0.1:3456
          extender.httpTimeout: 100ms
          extender.predicateVerb: predicate


*/
