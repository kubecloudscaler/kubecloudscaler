/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

// BuildHandlerChain links handlers together using SetNext() in the order they are provided
// and returns the first handler as the chain entry point.
//
// Parameters:
//   - handlers: Ordered list of handlers to execute
//
// Returns:
//   - Handler: The first handler in the chain (nil if no handlers provided)
func BuildHandlerChain(handlers ...Handler) Handler {
	if len(handlers) == 0 {
		return nil
	}

	// Link handlers together using SetNext pattern
	for i := range len(handlers) - 1 {
		handlers[i].SetNext(handlers[i+1])
	}

	return handlers[0]
}
