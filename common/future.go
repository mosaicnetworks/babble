/*
Copyright 2017 Mosaic Networks Ltd

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
package common

// Future is used to represent an action that may occur in the future.
type Future interface {
	// Error blocks until the future arrives and then
	// returns the error status of the future.
	// This may be called any number of times - all
	// calls will return the same value.
	// Note that it is not OK to call this method
	// twice concurrently on the same Future instance.
	Error() error
}
