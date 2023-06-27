// Copyright 2023 Tamás Gulácsi. All rights reserved.

package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestUnmarshal(t *testing.T) {
	for _, tC := range []struct {
		Src      string
		MantisID string
	}{

		{Src: `{"fields":{
			"customfield_15902":"12345",
			"customfield_num":1, 
			"customfield_string":"s", 
			"customfield_arr":["1",2]
		}}`, MantisID: "12345"},

		{Src: b64decode(`eyJleHBhbmQiOiJyZW5kZXJlZEZpZWxkcyxuYW1lcyxzY2hlbWEsb3BlcmF0aW9ucyxlZGl0bWV0YSxjaGFuZ2Vsb2csdmVyc2lvbmVkUmVwcmVzZW50YXRpb25zIiwiaWQiOiIzMTgxMDQiLCJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9pc3N1ZS8zMTgxMDQiLCJrZXkiOiJJTkNJREVOVC0zMDM1NSIsImZpZWxkcyI6eyJwYXJlbnQiOnsiaWQiOiIzMTgxMDMiLCJrZXkiOiJJTkNJREVOVC0zMDM1NCIsInNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2lzc3VlLzMxODEwMyIsImZpZWxkcyI6eyJzdW1tYXJ5IjoiTUFOVElTIHRlc3p0IDIwMjMuMDYuMDYuIiwic3RhdHVzIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL3N0YXR1cy8xMjcwNCIsImRlc2NyaXB0aW9uIjoiIiwiaWNvblVybCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L2ltYWdlcy9pY29ucy9zdGF0dXNlcy9nZW5lcmljLnBuZyIsIm5hbWUiOiJMMiBpbnZvbHZlZCIsImlkIjoiMTI3MDQiLCJzdGF0dXNDYXRlZ29yeSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9zdGF0dXNjYXRlZ29yeS80IiwiaWQiOjQsImtleSI6ImluZGV0ZXJtaW5hdGUiLCJjb2xvck5hbWUiOiJpbnByb2dyZXNzIiwibmFtZSI6IkluIFByb2dyZXNzIn19LCJwcmlvcml0eSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9wcmlvcml0eS8xMDcwMyIsImljb25VcmwiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9pbWFnZXMvaWNvbnMvcHJpb3JpdGllcy9sb3cuc3ZnIiwibmFtZSI6IlA0IEFMQUNTT05ZIiwiaWQiOiIxMDcwMyJ9LCJpc3N1ZXR5cGUiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvaXNzdWV0eXBlLzEwNjAxIiwiaWQiOiIxMDYwMSIsImRlc2NyaXB0aW9uIjoiRm9yIHN5c3RlbSBvdXRhZ2VzIG9yIGluY2lkZW50cy4gQ3JlYXRlZCBieSBKSVJBIFNlcnZpY2UgRGVzay4iLCJpY29uVXJsIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3ZpZXdhdmF0YXI/c2l6ZT14c21hbGwmYXZhdGFySWQ9MTE2MDQmYXZhdGFyVHlwZT1pc3N1ZXR5cGUiLCJuYW1lIjoiSW5jaWRlbnMiLCJzdWJ0YXNrIjpmYWxzZSwiYXZhdGFySWQiOjExNjA0fX19LCJyZXNvbHV0aW9uIjpudWxsLCJjdXN0b21maWVsZF8xMDUwMCI6bnVsbCwibGFzdFZpZXdlZCI6bnVsbCwiYWdncmVnYXRldGltZW9yaWdpbmFsZXN0aW1hdGUiOm51bGwsImlzc3VlbGlua3MiOltdLCJhc3NpZ25lZSI6bnVsbCwiY3VzdG9tZmllbGRfMTA2MDAiOm51bGwsInN1YnRhc2tzIjpbXSwiY3VzdG9tZmllbGRfMTE4MDAiOm51bGwsInZvdGVzIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2lzc3VlL0lOQ0lERU5ULTMwMzU1L3ZvdGVzIiwidm90ZXMiOjAsImhhc1ZvdGVkIjpmYWxzZX0sIndvcmtsb2ciOnsic3RhcnRBdCI6MCwibWF4UmVzdWx0cyI6MjAsInRvdGFsIjowLCJ3b3JrbG9ncyI6W119LCJpc3N1ZXR5cGUiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvaXNzdWV0eXBlLzEwMjAyIiwiaWQiOiIxMDIwMiIsImRlc2NyaXB0aW9uIjoiQSBmZWxhZGF0IGFsZmVsYWRhdGEuIiwiaWNvblVybCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS92aWV3YXZhdGFyP3NpemU9eHNtYWxsJmF2YXRhcklkPTEwMzE2JmF2YXRhclR5cGU9aXNzdWV0eXBlIiwibmFtZSI6IkFsZmVsYWRhdCIsInN1YnRhc2siOnRydWUsImF2YXRhcklkIjoxMDMxNn0sImN1c3RvbWZpZWxkXzExOTAxIjpudWxsLCJ0aW1ldHJhY2tpbmciOnt9LCJjdXN0b21maWVsZF8xNTAxNSI6W3siYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9XSwiY3VzdG9tZmllbGRfMTUwMTYiOlt7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9XSwiY3VzdG9tZmllbGRfMTUwMTQiOlt7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfV0sImVudmlyb25tZW50IjpudWxsLCJkdWVkYXRlIjoiMjAyMy0wNi0yMiIsImN1c3RvbWZpZWxkXzE0NzExIjpudWxsLCJjdXN0b21maWVsZF8xNDgzMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MTIiOm51bGwsImN1c3RvbWZpZWxkXzE0ODMzIjpudWxsLCJjdXN0b21maWVsZF8xNDgzMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MzEiOm51bGwsImN1c3RvbWZpZWxkXzE0NzEwIjpudWxsLCJjdXN0b21maWVsZF8xMDEwNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjUiOm51bGwsImN1c3RvbWZpZWxkXzE0NzA0IjpudWxsLCJjdXN0b21maWVsZF8xMDEwNSI6bnVsbCwiY3VzdG9tZmllbGRfMTI0MDMiOm51bGwsImN1c3RvbWZpZWxkXzE0ODI2IjpudWxsLCJjdXN0b21maWVsZF8xNDcwNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjMiOm51bGwsImN1c3RvbWZpZWxkXzE0NzAyIjpudWxsLCJjdXN0b21maWVsZF8xMjQwNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MDMiOm51bGwsImN1c3RvbWZpZWxkXzE0ODI0IjpudWxsLCJjdXN0b21maWVsZF8xNDcwOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjkiOm51bGwsImN1c3RvbWZpZWxkXzEyNDA3IjpudWxsLCJjdXN0b21maWVsZF8xNDcwOSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjciOm51bGwsImN1c3RvbWZpZWxkXzE0NzA2IjpudWxsLCJjdXN0b21maWVsZF8xNDcwNyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjgiOm51bGwsImN1c3RvbWZpZWxkXzEwMTAwIjpudWxsLCJjdXN0b21maWVsZF8xNDgyMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MDAiOm51bGwsImN1c3RvbWZpZWxkXzEwMTAxIjpudWxsLCJjdXN0b21maWVsZF8xNDcwMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjIiOm51bGwsImN1c3RvbWZpZWxkXzEwMTAyIjpudWxsLCJjdXN0b21maWVsZF8xMjQwMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MjAiOm51bGwsImN1c3RvbWZpZWxkXzE0ODE0Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE4MzU3IiwidmFsdWUiOiJOZW0iLCJpZCI6IjE4MzU3IiwiZGlzYWJsZWQiOmZhbHNlfSwiY3VzdG9tZmllbGRfMTQ4MTUiOm51bGwsImN1c3RvbWZpZWxkXzE0ODEyIjpudWxsLCJjdXN0b21maWVsZF8xNTkwMiI6IjE1Mjc5IiwiY3VzdG9tZmllbGRfMTQ4MTMiOm51bGwsImN1c3RvbWZpZWxkXzE0ODE4Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE4MzI2IiwidmFsdWUiOiJOZW0iLCJpZCI6IjE4MzI2IiwiZGlzYWJsZWQiOmZhbHNlfSwidGltZWVzdGltYXRlIjpudWxsLCJjdXN0b21maWVsZF8xNDgxOSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xODM1NSIsInZhbHVlIjoiTmVtIiwiaWQiOiIxODM1NSIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE0ODE2IjpudWxsLCJjdXN0b21maWVsZF8xNDgxNyI6IjAiLCJzdGF0dXMiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvc3RhdHVzLzEwNDIxIiwiZGVzY3JpcHRpb24iOiJUaGlzIHdhcyBhdXRvLWdlbmVyYXRlZCBieSBKSVJBIFNlcnZpY2UgRGVzayBkdXJpbmcgd29ya2Zsb3cgaW1wb3J0IiwiaWNvblVybCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L2ltYWdlcy9pY29ucy9zdGF0dXNfZ2VuZXJpYy5naWYiLCJuYW1lIjoiTnlpdMOhcyIsImlkIjoiMTA0MjEiLCJzdGF0dXNDYXRlZ29yeSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9zdGF0dXNjYXRlZ29yeS8yIiwiaWQiOjIsImtleSI6Im5ldyIsImNvbG9yTmFtZSI6ImRlZmF1bHQiLCJuYW1lIjoiVG8gRG8ifX0sImN1c3RvbWZpZWxkXzE0ODEwIjpudWxsLCJjdXN0b21maWVsZF8xNTkwMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MTEiOm51bGwsImN1c3RvbWZpZWxkXzExMzAxIjpudWxsLCJjdXN0b21maWVsZF8xMTMwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MDciOm51bGwsImFnZ3JlZ2F0ZXRpbWVlc3RpbWF0ZSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MDgiOm51bGwsImN1c3RvbWZpZWxkXzE0ODA1IjpudWxsLCJjdXN0b21maWVsZF8xNDgwOSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xODM0MCIsInZhbHVlIjoiTmVtIE9DIiwiaWQiOiIxODM0MCIsImRpc2FibGVkIjpmYWxzZX0sImNyZWF0b3IiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvdXNlcj91c2VybmFtZT1raXNzYmFsIiwibmFtZSI6Imtpc3NiYWwiLCJrZXkiOiJKSVJBVVNFUjE5NTQ3IiwiZW1haWxBZGRyZXNzIjoiS2lzcy5CYWxhenNAYWVnb24uaHUiLCJhdmF0YXJVcmxzIjp7IjQ4eDQ4IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/b3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMjR4MjQiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjE2eDE2IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT14c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMzJ4MzIiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPW1lZGl1bSZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYifSwiZGlzcGxheU5hbWUiOiJLaXNzLCBCYWzDoXpzIiwiYWN0aXZlIjp0cnVlLCJ0aW1lWm9uZSI6IkV1cm9wZS9CdWRhcGVzdCJ9LCJjdXN0b21maWVsZF8xMjYxNSI6bnVsbCwidGltZXNwZW50IjpudWxsLCJhZ2dyZWdhdGV0aW1lc3BlbnQiOm51bGwsImN1c3RvbWZpZWxkXzExNDAxIjpudWxsLCJjdXN0b21maWVsZF8xMTQwMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ5MDIiOm51bGwsImN1c3RvbWZpZWxkXzE0OTAzIjpudWxsLCJjdXN0b21maWVsZF8xNDkwMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ5MDEiOm51bGwsImN1c3RvbWZpZWxkXzE0OTA0IjpudWxsLCJ3b3JrcmF0aW8iOi0xLCJjdXN0b21maWVsZF8xMDMwMCI6Int9IiwiY3VzdG9tZmllbGRfMTAzMDEiOm51bGwsImN1c3RvbWZpZWxkXzEyNzEyIjpudWxsLCJjdXN0b21maWVsZF8xMzgwMSI6bnVsbCwiY3VzdG9tZmllbGRfMTI3MTEiOm51bGwsImN1c3RvbWZpZWxkXzEzODAzIjpudWxsLCJjdXN0b21maWVsZF8xMzgwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTM4MDQiOm51bGwsImN1c3RvbWZpZWxkXzExNTAwIjpudWxsLCJjdXN0b21maWVsZF8xMjcxMCI6bnVsbCwiY3VzdG9tZmllbGRfMTI3MDUiOm51bGwsImN1c3RvbWZpZWxkXzEyNzA0IjpudWxsLCJjdXN0b21maWVsZF8xMjcwNyI6bnVsbCwiY3VzdG9tZmllbGRfMTI3MDYiOm51bGwsImN1c3RvbWZpZWxkXzEyNzA5IjpudWxsLCJjdXN0b21maWVsZF8xMjcwOCI6bnVsbCwiY3VzdG9tZmllbGRfMTI4MTEiOm51bGwsImN1c3RvbWZpZWxkXzEyODEwIjpudWxsLCJjdXN0b21maWVsZF8xNDQzMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMTEiOm51bGwsImN1c3RvbWZpZWxkXzE0NDMzIjpudWxsLCJjdXN0b21maWVsZF8xNDQzMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMTAiOm51bGwsImN1c3RvbWZpZWxkXzE0NDMxIjpudWxsLCJjdXN0b21maWVsZF8xNDMxNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MzciOm51bGwsImN1c3RvbWZpZWxkXzE0NDM1IjpudWxsLCJjdXN0b21maWVsZF8xNDMwOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MjkiOm51bGwsImN1c3RvbWZpZWxkXzE0MzA5IjpudWxsLCJjdXN0b21maWVsZF8xNDMwNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MjciOm51bGwsImN1c3RvbWZpZWxkXzE0MzA3Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE2OTAwIiwidmFsdWUiOiIzcmQgUGFydGllcyIsImlkIjoiMTY5MDAiLCJkaXNhYmxlZCI6ZmFsc2UsImNoaWxkIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE2OTM4IiwidmFsdWUiOiJVbm8tU29mdCBTesOhbcOtdMOhc3RlY2huaWthaSBLZnQuIiwiaWQiOiIxNjkzOCIsImRpc2FibGVkIjpmYWxzZX19LCJjdXN0b21maWVsZF8xNDQyOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMDAiOm51bGwsImN1c3RvbWZpZWxkXzEyMDAwIjpudWxsLCJjdXN0b21maWVsZF8xNDQyMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMDEiOm51bGwsImN1c3RvbWZpZWxkXzE0NDIyIjpudWxsLCJjdXN0b21maWVsZF8xNDQyMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMDQiOm51bGwsImN1c3RvbWZpZWxkXzE0NDI1Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE3NTQzIiwidmFsdWUiOiJBZWdvbiBNYWd5YXJvcm9zesOhZyBacnQuIiwiaWQiOiIxNzU0MyIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE0MzA1IjpudWxsLCJjdXN0b21maWVsZF8xNDQyNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMDIiOm51bGwsImN1c3RvbWZpZWxkXzE0NDIzIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE3NTM0IiwidmFsdWUiOiJOb3Jtw6FsIiwiaWQiOiIxNzUzNCIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE2NjAyIjpudWxsLCJsYWJlbHMiOltdLCJjdXN0b21maWVsZF8xNDMwMyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MjQiOm51bGwsImN1c3RvbWZpZWxkXzE0NDE4IjpudWxsLCJjdXN0b21maWVsZF8xNDQxOSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MTYiOm51bGwsImNvbXBvbmVudHMiOltdLCJjdXN0b21maWVsZF8xNDQxMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MTEiOm51bGwsImN1c3RvbWZpZWxkXzE1NTAwIjpudWxsLCJjdXN0b21maWVsZF8xNDQxNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MTUiOm51bGwsImN1c3RvbWZpZWxkXzE0NDEyIjpudWxsLCJjdXN0b21maWVsZF8xNDQxMyI6bnVsbCwiY3VzdG9tZmllbGRfMTAwNDkiOm51bGwsImN1c3RvbWZpZWxkXzE0NDA3IjpudWxsLCJjdXN0b21maWVsZF8xNDQwOCI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xNzUwMCIsInZhbHVlIjoiTm8iLCJpZCI6IjE3NTAwIiwiZGlzYWJsZWQiOmZhbHNlfSwiY3VzdG9tZmllbGRfMTQ0MDUiOm51bGwsImN1c3RvbWZpZWxkXzE0NDA2IjpudWxsLCJjdXN0b21maWVsZF8xNDQwOSI6bnVsbCwiY3VzdG9tZmllbGRfMTAwNDAiOm51bGwsImN1c3RvbWZpZWxkXzEwMDQxIjpudWxsLCJjdXN0b21maWVsZF8xMDA0MiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MDAiOm51bGwsInJlcG9ydGVyIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL3VzZXI/dXNlcm5hbWU9a2lzc2JhbCIsIm5hbWUiOiJraXNzYmFsIiwia2V5IjoiSklSQVVTRVIxOTU0NyIsImVtYWlsQWRkcmVzcyI6Iktpc3MuQmFsYXpzQGFlZ29uLmh1IiwiYXZhdGFyVXJscyI6eyI0OHg0OCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP293bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjI0eDI0IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT1zbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIxNngxNiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9eHNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjMyeDMyIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT1tZWRpdW0mb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2In0sImRpc3BsYXlOYW1lIjoiS2lzcywgQmFsw6F6cyIsImFjdGl2ZSI6dHJ1ZSwidGltZVpvbmUiOiJFdXJvcGUvQnVkYXBlc3QifSwiY3VzdG9tZmllbGRfMTAwNDMiOm51bGwsImN1c3RvbWZpZWxkXzEwMDQ0IjpudWxsLCJjdXN0b21maWVsZF8xNDY0MCI6bnVsbCwiY3VzdG9tZmllbGRfMTAwNDUiOm51bGwsImN1c3RvbWZpZWxkXzE0NDAzIjpudWxsLCJjdXN0b21maWVsZF8xMDA0NiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0MDQiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvY3VzdG9tRmllbGRPcHRpb24vMTc1MTciLCJ2YWx1ZSI6Ik5vIiwiaWQiOiIxNzUxNyIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzEwMDQ3IjpudWxsLCJjdXN0b21maWVsZF8xNDQwMSI6bnVsbCwiY3VzdG9tZmllbGRfMTAwNDgiOm51bGwsImN1c3RvbWZpZWxkXzE0NDAyIjpudWxsLCJjdXN0b21maWVsZF8xMDAzOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MzgiOm51bGwsImN1c3RvbWZpZWxkXzEwMDM5IjpudWxsLCJjdXN0b21maWVsZF8xNDYzOSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NzgiOlt7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX1dLCJjdXN0b21maWVsZF8xNDYzNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NzkiOlt7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfV0sImN1c3RvbWZpZWxkXzE0NjM3IjpudWxsLCJwcm9ncmVzcyI6eyJwcm9ncmVzcyI6MCwidG90YWwiOjB9LCJjdXN0b21maWVsZF8xMDAzMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MzAiOm51bGwsImN1c3RvbWZpZWxkXzEwMDMxIjpudWxsLCJjdXN0b21maWVsZF8xNDg3MyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MzEiOm51bGwsImN1c3RvbWZpZWxkXzE0NTEwIjpudWxsLCJwcm9qZWN0Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL3Byb2plY3QvMTQ1MDIiLCJpZCI6IjE0NTAyIiwia2V5IjoiSU5DSURFTlQiLCJuYW1lIjoiSW5jaWRlbnQgTWFuYWdlbWVudCIsInByb2plY3RUeXBlS2V5Ijoic2VydmljZV9kZXNrIiwiYXZhdGFyVXJscyI6eyI0OHg0OCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS9wcm9qZWN0YXZhdGFyP3BpZD0xNDUwMiZhdmF0YXJJZD0xMDMzMSIsIjI0eDI0IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3Byb2plY3RhdmF0YXI/c2l6ZT1zbWFsbCZwaWQ9MTQ1MDImYXZhdGFySWQ9MTAzMzEiLCIxNngxNiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS9wcm9qZWN0YXZhdGFyP3NpemU9eHNtYWxsJnBpZD0xNDUwMiZhdmF0YXJJZD0xMDMzMSIsIjMyeDMyIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3Byb2plY3RhdmF0YXI/c2l6ZT1tZWRpdW0mcGlkPTE0NTAyJmF2YXRhcklkPTEwMzMxIn19LCJjdXN0b21maWVsZF8xMDAzMiI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMzMiOm51bGwsImN1c3RvbWZpZWxkXzEwMDM0IjpudWxsLCJjdXN0b21maWVsZF8xNDg3NiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MzQiOm51bGwsImN1c3RvbWZpZWxkXzE0NTEzIjpudWxsLCJjdXN0b21maWVsZF8xMDAzNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NzciOm51bGwsImN1c3RvbWZpZWxkXzE0NjM1IjpudWxsLCJjdXN0b21maWVsZF8xMDAzNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NzQiOm51bGwsImN1c3RvbWZpZWxkXzE0NjMyIjpudWxsLCJjdXN0b21maWVsZF8xNDUxMSI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMzciOm51bGwsImN1c3RvbWZpZWxkXzE0ODc1IjpudWxsLCJjdXN0b21maWVsZF8xNDUxMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MzMiOm51bGwsImN1c3RvbWZpZWxkXzE1NjAxIjpudWxsLCJjdXN0b21maWVsZF8xNDYyNyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ1MDYiOm51bGwsImN1c3RvbWZpZWxkXzEwMDI4IjpudWxsLCJjdXN0b21maWVsZF8xNDUwNyI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMjkiOm51bGwsImN1c3RvbWZpZWxkXzE0NTA0IjpudWxsLCJjdXN0b21maWVsZF8xNDYyNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MjYiOm51bGwsImN1c3RvbWZpZWxkXzE0NTA1IjpudWxsLCJjdXN0b21maWVsZF8xNDUwOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MjkiOm51bGwsInJlc29sdXRpb25kYXRlIjpudWxsLCJjdXN0b21maWVsZF8xNDUwOSI6bnVsbCwid2F0Y2hlcyI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9pc3N1ZS9JTkNJREVOVC0zMDM1NS93YXRjaGVycyIsIndhdGNoQ291bnQiOjEsImlzV2F0Y2hpbmciOmZhbHNlfSwiY3VzdG9tZmllbGRfMTQ4NjEiOm51bGwsImN1c3RvbWZpZWxkXzE0NjIwIjpudWxsLCJjdXN0b21maWVsZF8xNDUwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MjQiOm51bGwsImN1c3RvbWZpZWxkXzE0NTAzIjpudWxsLCJjdXN0b21maWVsZF8xNDUwMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MjEiOm51bGwsImN1c3RvbWZpZWxkXzE0NTAxIjpudWxsLCJjdXN0b21maWVsZF8xNDg1OCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MTYiOm51bGwsImN1c3RvbWZpZWxkXzEwMDE3IjpudWxsLCJjdXN0b21maWVsZF8xNDg1OSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTYiOm51bGwsImN1c3RvbWZpZWxkXzE0NjE0IjpudWxsLCJjdXN0b21maWVsZF8xNDYxNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTciOm51bGwsImN1c3RvbWZpZWxkXzE0NjE4IjpudWxsLCJjdXN0b21maWVsZF8xNDYxOSI6bnVsbCwidXBkYXRlZCI6IjIwMjMtMDYtMjZUMTM6MjY6NDkuNTE1KzAyMDAiLCJ0aW1lb3JpZ2luYWxlc3RpbWF0ZSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTAiOm51bGwsImRlc2NyaXB0aW9uIjoiVEVTWlQiLCJjdXN0b21maWVsZF8xNDg1MSI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMTAiOltdLCJjdXN0b21maWVsZF8xMDAxMSI6bnVsbCwiY3VzdG9tZmllbGRfMTExMDAiOnsiaWQiOiIxMzgiLCJuYW1lIjoiSWTFkSBhIG1lZ29sZMOhc2lnIiwiX2xpbmtzIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L3NlcnZpY2VkZXNrYXBpL3JlcXVlc3QvMzE4MTA0L3NsYS8xMzgifSwiY29tcGxldGVkQ3ljbGVzIjpbXX0sImN1c3RvbWZpZWxkXzEwMDEyIjpudWxsLCJjdXN0b21maWVsZF8xMTEwMSI6eyJpZCI6IjEzOSIsIm5hbWUiOiJJZMWRIGF6IGVsc8WRIHbDoWxhc3ppZyIsIl9saW5rcyI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9zZXJ2aWNlZGVza2FwaS9yZXF1ZXN0LzMxODEwNC9zbGEvMTM5In0sImNvbXBsZXRlZEN5Y2xlcyI6W10sIm9uZ29pbmdDeWNsZSI6eyJzdGFydFRpbWUiOnsiaXNvODYwMSI6IjIwMjMtMDYtMDZUMTI6MTc6MTcrMDIwMCIsImppcmEiOiIyMDIzLTA2LTA2VDEyOjE3OjE3LjI5MyswMjAwIiwiZnJpZW5kbHkiOiIyMDIzLjA2LjA2LiAxMjoxNyIsImVwb2NoTWlsbGlzIjoxNjg2MDQ2NjM3MjkzfSwiYnJlYWNoVGltZSI6eyJpc284NjAxIjoiMjAyMy0wNi0xM1QxMToxNzoxNyswMjAwIiwiamlyYSI6IjIwMjMtMDYtMTNUMTE6MTc6MTcuMjkzKzAyMDAiLCJmcmllbmRseSI6IjIwMjMuMDYuMTMuIDExOjE3IiwiZXBvY2hNaWxsaXMiOjE2ODY2NDc4MzcyOTN9LCJicmVhY2hlZCI6dHJ1ZSwicGF1c2VkIjpmYWxzZSwid2l0aGluQ2FsZW5kYXJIb3VycyI6dHJ1ZSwiZ29hbER1cmF0aW9uIjp7Im1pbGxpcyI6MTQ0MDAwMDAwLCJmcmllbmRseSI6IjQwaCJ9LCJlbGFwc2VkVGltZSI6eyJtaWxsaXMiOjQzOTUwMDM1NiwiZnJpZW5kbHkiOiIxMjJoIDVtIn0sInJlbWFpbmluZ1RpbWUiOnsibWlsbGlzIjotMjk1NTAwMzU2LCJmcmllbmRseSI6Ii04MmggNW0ifX19LCJjdXN0b21maWVsZF8xNDYxMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTQiOm51bGwsImN1c3RvbWZpZWxkXzEwMDEzIjpudWxsLCJjdXN0b21maWVsZF8xMTEwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTUiOm51bGwsImN1c3RvbWZpZWxkXzE0NjEzIjpudWxsLCJjdXN0b21maWVsZF8xMDAxNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NTIiOm51bGwsImN1c3RvbWZpZWxkXzE0NjEwIjpudWxsLCJjdXN0b21maWVsZF8xMDAxNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MTEiOm51bGwsImN1c3RvbWZpZWxkXzE0ODUzIjpudWxsLCJjdXN0b21maWVsZF8xMDAwNSI6IjJ8dTE5ODZvOiIsImN1c3RvbWZpZWxkXzE0NjA1IjpudWxsLCJjdXN0b21maWVsZF8xNDcyNiI6W3siYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX1dLCJjdXN0b21maWVsZF8xNDg0NyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MjciOm51bGwsImN1c3RvbWZpZWxkXzE0NjA2IjpudWxsLCJjdXN0b21maWVsZF8xNDg0OCI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xODM5NSIsInZhbHVlIjoiTmVtIiwiaWQiOiIxODM5NSIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzEwMDA3IjpudWxsLCJjdXN0b21maWVsZF8xNDYwMyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NDUiOm51bGwsImN1c3RvbWZpZWxkXzE0NzI0IjpudWxsLCJjdXN0b21maWVsZF8xMDAwOCI6W10sImN1c3RvbWZpZWxkXzE0NzI1IjpudWxsLCJjdXN0b21maWVsZF8xNDg0NiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MDQiOm51bGwsImN1c3RvbWZpZWxkXzEwMDA5IjpudWxsLCJjdXN0b21maWVsZF8xNDYwOSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MDciOm51bGwsImN1c3RvbWZpZWxkXzE0ODQ5IjpudWxsLCJjdXN0b21maWVsZF8xNDcyOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MDgiOm51bGwsImN1c3RvbWZpZWxkXzE0NzI5IjpbeyJhY3RpdmUiOnRydWV9LHsiYWN0aXZlIjp0cnVlfSx7ImFjdGl2ZSI6dHJ1ZX0seyJhY3RpdmUiOnRydWV9XSwic3VtbWFyeSI6Ik1BTlRJUyBURVNaVDEgMjAyMy4wNi4wNi4iLCJjdXN0b21maWVsZF8xNDg0MCI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMDAiOm51bGwsImN1c3RvbWZpZWxkXzE0NjAxIjpudWxsLCJjdXN0b21maWVsZF8xNDg0MyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MjIiOm51bGwsImN1c3RvbWZpZWxkXzE0NzIzIjpudWxsLCJjdXN0b21maWVsZF8xNDYwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4NDQiOm51bGwsImN1c3RvbWZpZWxkXzE0ODQxIjpudWxsLCJjdXN0b21maWVsZF8xNDcyMCI6bnVsbCwiY3VzdG9tZmllbGRfMTAwMDQiOm51bGwsImN1c3RvbWZpZWxkXzE0ODQyIjpudWxsLCJjdXN0b21maWVsZF8xNDcyMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ2MDAiOm51bGwsImN1c3RvbWZpZWxkXzEzNTA0IjpudWxsLCJjdXN0b21maWVsZF8xNTgwNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MTUiOm51bGwsImN1c3RvbWZpZWxkXzE0ODM2IjpudWxsLCJjdXN0b21maWVsZF8xNTgwNSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MTYiOm51bGwsImN1c3RvbWZpZWxkXzE0ODM3IjpudWxsLCJjdXN0b21maWVsZF8xNDcxMyI6bnVsbCwiY3VzdG9tZmllbGRfMTM1MDUiOm51bGwsImN1c3RvbWZpZWxkXzE1ODAzIjpudWxsLCJjdXN0b21maWVsZF8xNDcxNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MzUiOm51bGwsImN1c3RvbWZpZWxkXzE0NzE5IjpudWxsLCJjdXN0b21maWVsZF8xNDgzOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ3MTciOm51bGwsImN1c3RvbWZpZWxkXzE1ODA2IjpudWxsLCJjdXN0b21maWVsZF8xNDcxOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ4MzkiOm51bGwsImNvbW1lbnQiOnsiY29tbWVudHMiOlt7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2lzc3VlLzMxODEwNC9jb21tZW50LzQxODA0NSIsImlkIjoiNDE4MDQ1IiwiYXV0aG9yIjp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL3VzZXI/dXNlcm5hbWU9a2lzc2JhbCIsIm5hbWUiOiJraXNzYmFsIiwia2V5IjoiSklSQVVTRVIxOTU0NyIsImVtYWlsQWRkcmVzcyI6Iktpc3MuQmFsYXpzQGFlZ29uLmh1IiwiYXZhdGFyVXJscyI6eyI0OHg0OCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP293bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjI0eDI0IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT1zbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIxNngxNiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9eHNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjMyeDMyIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT1tZWRpdW0mb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2In0sImRpc3BsYXlOYW1lIjoiS2lzcywgQmFsw6F6cyIsImFjdGl2ZSI6dHJ1ZSwidGltZVpvbmUiOiJFdXJvcGUvQnVkYXBlc3QifSwiYm9keSI6InRlc3p0IGtvbW1lbnQiLCJ1cGRhdGVBdXRob3IiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvdXNlcj91c2VybmFtZT1raXNzYmFsIiwibmFtZSI6Imtpc3NiYWwiLCJrZXkiOiJKSVJBVVNFUjE5NTQ3IiwiZW1haWxBZGRyZXNzIjoiS2lzcy5CYWxhenNAYWVnb24uaHUiLCJhdmF0YXJVcmxzIjp7IjQ4eDQ4IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/b3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMjR4MjQiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjE2eDE2IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT14c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMzJ4MzIiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPW1lZGl1bSZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYifSwiZGlzcGxheU5hbWUiOiJLaXNzLCBCYWzDoXpzIiwiYWN0aXZlIjp0cnVlLCJ0aW1lWm9uZSI6IkV1cm9wZS9CdWRhcGVzdCJ9LCJjcmVhdGVkIjoiMjAyMy0wNi0yNlQxMzowNzo1Ny43NDMrMDIwMCIsInVwZGF0ZWQiOiIyMDIzLTA2LTI2VDEzOjA3OjU3Ljc0MyswMjAwIn0seyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9pc3N1ZS8zMTgxMDQvY29tbWVudC80MTgwNDciLCJpZCI6IjQxODA0NyIsImF1dGhvciI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi91c2VyP3VzZXJuYW1lPWtpc3NiYWwiLCJuYW1lIjoia2lzc2JhbCIsImtleSI6IkpJUkFVU0VSMTk1NDciLCJlbWFpbEFkZHJlc3MiOiJLaXNzLkJhbGF6c0BhZWdvbi5odSIsImF2YXRhclVybHMiOnsiNDh4NDgiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9vd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIyNHgyNCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMTZ4MTYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXhzbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIzMngzMiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9bWVkaXVtJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiJ9LCJkaXNwbGF5TmFtZSI6Iktpc3MsIEJhbMOhenMiLCJhY3RpdmUiOnRydWUsInRpbWVab25lIjoiRXVyb3BlL0J1ZGFwZXN0In0sImJvZHkiOiJ0ZXN6dCBrb21tZW50IDIiLCJ1cGRhdGVBdXRob3IiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvdXNlcj91c2VybmFtZT1raXNzYmFsIiwibmFtZSI6Imtpc3NiYWwiLCJrZXkiOiJKSVJBVVNFUjE5NTQ3IiwiZW1haWxBZGRyZXNzIjoiS2lzcy5CYWxhenNAYWVnb24uaHUiLCJhdmF0YXJVcmxzIjp7IjQ4eDQ4IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/b3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMjR4MjQiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjE2eDE2IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT14c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMzJ4MzIiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPW1lZGl1bSZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYifSwiZGlzcGxheU5hbWUiOiJLaXNzLCBCYWzDoXpzIiwiYWN0aXZlIjp0cnVlLCJ0aW1lWm9uZSI6IkV1cm9wZS9CdWRhcGVzdCJ9LCJjcmVhdGVkIjoiMjAyMy0wNi0yNlQxMzowOTowMS40NjArMDIwMCIsInVwZGF0ZWQiOiIyMDIzLTA2LTI2VDEzOjA5OjAxLjQ2MCswMjAwIn0seyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9pc3N1ZS8zMTgxMDQvY29tbWVudC80MTgwNTAiLCJpZCI6IjQxODA1MCIsImF1dGhvciI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi91c2VyP3VzZXJuYW1lPWtpc3NiYWwiLCJuYW1lIjoia2lzc2JhbCIsImtleSI6IkpJUkFVU0VSMTk1NDciLCJlbWFpbEFkZHJlc3MiOiJLaXNzLkJhbGF6c0BhZWdvbi5odSIsImF2YXRhclVybHMiOnsiNDh4NDgiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9vd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIyNHgyNCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMTZ4MTYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXhzbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIzMngzMiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9bWVkaXVtJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiJ9LCJkaXNwbGF5TmFtZSI6Iktpc3MsIEJhbMOhenMiLCJhY3RpdmUiOnRydWUsInRpbWVab25lIjoiRXVyb3BlL0J1ZGFwZXN0In0sImJvZHkiOiJbXnRlc3p0LnR4dF0iLCJ1cGRhdGVBdXRob3IiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvdXNlcj91c2VybmFtZT1raXNzYmFsIiwibmFtZSI6Imtpc3NiYWwiLCJrZXkiOiJKSVJBVVNFUjE5NTQ3IiwiZW1haWxBZGRyZXNzIjoiS2lzcy5CYWxhenNAYWVnb24uaHUiLCJhdmF0YXJVcmxzIjp7IjQ4eDQ4IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/b3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMjR4MjQiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXNtYWxsJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiIsIjE2eDE2IjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvc2VjdXJlL3VzZXJhdmF0YXI/c2l6ZT14c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMzJ4MzIiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPW1lZGl1bSZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYifSwiZGlzcGxheU5hbWUiOiJLaXNzLCBCYWzDoXpzIiwiYWN0aXZlIjp0cnVlLCJ0aW1lWm9uZSI6IkV1cm9wZS9CdWRhcGVzdCJ9LCJjcmVhdGVkIjoiMjAyMy0wNi0yNlQxMzoyMzowMS4xMTQrMDIwMCIsInVwZGF0ZWQiOiIyMDIzLTA2LTI2VDEzOjIzOjAxLjExNCswMjAwIn0seyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9pc3N1ZS8zMTgxMDQvY29tbWVudC80MTgwNTIiLCJpZCI6IjQxODA1MiIsImF1dGhvciI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi91c2VyP3VzZXJuYW1lPWtpc3NiYWwiLCJuYW1lIjoia2lzc2JhbCIsImtleSI6IkpJUkFVU0VSMTk1NDciLCJlbWFpbEFkZHJlc3MiOiJLaXNzLkJhbGF6c0BhZWdvbi5odSIsImF2YXRhclVybHMiOnsiNDh4NDgiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9vd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIyNHgyNCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMTZ4MTYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXhzbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIzMngzMiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9bWVkaXVtJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiJ9LCJkaXNwbGF5TmFtZSI6Iktpc3MsIEJhbMOhenMiLCJhY3RpdmUiOnRydWUsInRpbWVab25lIjoiRXVyb3BlL0J1ZGFwZXN0In0sImJvZHkiOiJzenRkYXNzZGFzYSIsInVwZGF0ZUF1dGhvciI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi91c2VyP3VzZXJuYW1lPWtpc3NiYWwiLCJuYW1lIjoia2lzc2JhbCIsImtleSI6IkpJUkFVU0VSMTk1NDciLCJlbWFpbEFkZHJlc3MiOiJLaXNzLkJhbGF6c0BhZWdvbi5odSIsImF2YXRhclVybHMiOnsiNDh4NDgiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9vd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIyNHgyNCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMTZ4MTYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXhzbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIzMngzMiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9bWVkaXVtJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiJ9LCJkaXNwbGF5TmFtZSI6Iktpc3MsIEJhbMOhenMiLCJhY3RpdmUiOnRydWUsInRpbWVab25lIjoiRXVyb3BlL0J1ZGFwZXN0In0sImNyZWF0ZWQiOiIyMDIzLTA2LTI2VDEzOjI2OjQ5LjUxNSswMjAwIiwidXBkYXRlZCI6IjIwMjMtMDYtMjZUMTM6MjY6NDkuNTE1KzAyMDAifV0sIm1heFJlc3VsdHMiOjQsInRvdGFsIjo0LCJzdGFydEF0IjowfSwiY3VzdG9tZmllbGRfMTUwMDAiOm51bGwsImN1c3RvbWZpZWxkXzE1MDAxIjpudWxsLCJmaXhWZXJzaW9ucyI6W10sImN1c3RvbWZpZWxkXzEwOTAwIjpudWxsLCJwcmlvcml0eSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9wcmlvcml0eS8xMDcwMyIsImljb25VcmwiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9pbWFnZXMvaWNvbnMvcHJpb3JpdGllcy9sb3cuc3ZnIiwibmFtZSI6IlA0IEFMQUNTT05ZIiwiaWQiOiIxMDcwMyJ9LCJ2ZXJzaW9ucyI6W10sImN1c3RvbWZpZWxkXzE0NDkxIjpudWxsLCJjdXN0b21maWVsZF8xNDQ5NCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0OTUiOm51bGwsImN1c3RvbWZpZWxkXzE0NDkyIjpudWxsLCJjdXN0b21maWVsZF8xNDQ5MyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0OTgiOm51bGwsImN1c3RvbWZpZWxkXzE0NDk5IjpudWxsLCJjdXN0b21maWVsZF8xNDQ5NiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0OTciOm51bGwsImN1c3RvbWZpZWxkXzE0NDgzIjpudWxsLCJjdXN0b21maWVsZF8xNDQ4NCI6bnVsbCwiY3VzdG9tZmllbGRfMTUzMzEiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvY3VzdG9tRmllbGRPcHRpb24vMTkxNzkiLCJ2YWx1ZSI6Ik5vcm3DoWwiLCJpZCI6IjE5MTc5IiwiZGlzYWJsZWQiOmZhbHNlfSwiY3VzdG9tZmllbGRfMTQ0ODEiOm51bGwsImN1c3RvbWZpZWxkXzE0NDgyIjpudWxsLCJhZ2dyZWdhdGVwcm9ncmVzcyI6eyJwcm9ncmVzcyI6MCwidG90YWwiOjB9LCJjdXN0b21maWVsZF8xNTIxOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NzIiOm51bGwsImN1c3RvbWZpZWxkXzE1MzIwIjpudWxsLCJjdXN0b21maWVsZF8xNDQ3MCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NzEiOm51bGwsImN1c3RvbWZpZWxkXzE0NDc2IjpudWxsLCJjdXN0b21maWVsZF8xNTMyMyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NzciOm51bGwsImN1c3RvbWZpZWxkXzE0NDc1IjpudWxsLCJjdXN0b21maWVsZF8xNTMyNiI6bnVsbCwiY3VzdG9tZmllbGRfMTUzMTkiOm51bGwsImN1c3RvbWZpZWxkXzE0NDYxIjpudWxsLCJjdXN0b21maWVsZF8xNDM0MCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NjIiOm51bGwsImN1c3RvbWZpZWxkXzE0MzQxIjpudWxsLCJjcmVhdGVkIjoiMjAyMy0wNi0wNlQxMjoxNzoxNy4yOTMrMDIwMCIsImN1c3RvbWZpZWxkXzE0NDY1IjpudWxsLCJjdXN0b21maWVsZF8xNDEwMiI6bnVsbCwiY3VzdG9tZmllbGRfMTQzNDQiOnsiaWQiOiIxMzUiLCJuYW1lIjoiRmVsaGFzem7DoWzDszogSmVneXrDoXLDoXMiLCJfbGlua3MiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3Qvc2VydmljZWRlc2thcGkvcmVxdWVzdC8zMTgxMDQvc2xhLzEzNSJ9LCJjb21wbGV0ZWRDeWNsZXMiOltdfSwiY3VzdG9tZmllbGRfMTQ0NjYiOm51bGwsImN1c3RvbWZpZWxkXzE0NDYzIjpudWxsLCJjdXN0b21maWVsZF8xNDEwMCI6bnVsbCwiY3VzdG9tZmllbGRfMTQzNDIiOnsiaWQiOiIxMzEiLCJuYW1lIjoiRmVsaGFzem7DoWzDszogSW5ha3Rpdml0w6FzIDIuIHdhcm4iLCJfbGlua3MiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3Qvc2VydmljZWRlc2thcGkvcmVxdWVzdC8zMTgxMDQvc2xhLzEzMSJ9LCJjb21wbGV0ZWRDeWNsZXMiOltdfSwiY3VzdG9tZmllbGRfMTQxMDEiOm51bGwsImN1c3RvbWZpZWxkXzE0NDY0IjpudWxsLCJjdXN0b21maWVsZF8xNDM0MyI6eyJpZCI6IjEzMiIsIm5hbWUiOiJGZWxoYXN6bsOhbMOzOiBJbmFrdGl2aXTDoXMgMS4gd2FybiIsIl9saW5rcyI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9zZXJ2aWNlZGVza2FwaS9yZXF1ZXN0LzMxODEwNC9zbGEvMTMyIn0sImNvbXBsZXRlZEN5Y2xlcyI6W119LCJjdXN0b21maWVsZF8xNTMxMSI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NjkiOm51bGwsImN1c3RvbWZpZWxkXzE1MzE3IjpudWxsLCJjdXN0b21maWVsZF8xNDQ2NyI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NjgiOm51bGwsImN1c3RvbWZpZWxkXzE1MzA5IjpudWxsLCJjdXN0b21maWVsZF8xNDMzOSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xNzM4NyIsInZhbHVlIjoiUG9ydMOhbCIsImlkIjoiMTczODciLCJkaXNhYmxlZCI6ZmFsc2V9LCJjdXN0b21maWVsZF8xNTMwOCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NTAiOm51bGwsImN1c3RvbWZpZWxkXzE0MzMwIjpudWxsLCJjdXN0b21maWVsZF8xNDQ1MSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xNzU2NyIsInZhbHVlIjoiTG93IiwiaWQiOiIxNzU2NyIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE0NDU0IjpudWxsLCJjdXN0b21maWVsZF8xNTMwMSI6bnVsbCwiY3VzdG9tZmllbGRfMTMwMDAiOm51bGwsImN1c3RvbWZpZWxkXzE0MzM0IjpudWxsLCJjdXN0b21maWVsZF8xNDQ1NSI6bnVsbCwiY3VzdG9tZmllbGRfMTUzMDIiOm51bGwsImN1c3RvbWZpZWxkXzE0MzMxIjpbeyJhY3RpdmUiOnRydWV9XSwiY3VzdG9tZmllbGRfMTQ0NTIiOm51bGwsImN1c3RvbWZpZWxkXzE0MzMyIjpudWxsLCJjdXN0b21maWVsZF8xNDQ1MyI6bnVsbCwiY3VzdG9tZmllbGRfMTUzMDAiOm51bGwsImN1c3RvbWZpZWxkXzE0MzM3IjpudWxsLCJjdXN0b21maWVsZF8xNDQ1OCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NTkiOm51bGwsImN1c3RvbWZpZWxkXzE0MzM4IjpudWxsLCJjdXN0b21maWVsZF8xNTMwNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NTYiOm51bGwsImN1c3RvbWZpZWxkXzE0MzM1IjpudWxsLCJjdXN0b21maWVsZF8xNDQ1NyI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMzYiOlt7Im5hbWUiOiJVbm8tU29mdCBTesOhbcOtdMOhc3RlY2huaWthaSBLZnQuIiwic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvZ3JvdXA/Z3JvdXBuYW1lPVVuby1Tb2Z0K1N6JUMzJUExbSVDMyVBRHQlQzMlQTFzdGVjaG5pa2FpK0tmdC4ifV0sImN1c3RvbWZpZWxkXzE1MzA0IjpudWxsLCJjdXN0b21maWVsZF8xNDQ0OSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xNzU1MiIsInZhbHVlIjoiTWlub3IiLCJpZCI6IjE3NTUyIiwiZGlzYWJsZWQiOmZhbHNlfSwic2VjdXJpdHkiOnsic2VsZiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3Jlc3QvYXBpLzIvc2VjdXJpdHlsZXZlbC8xMDIwMCIsImlkIjoiMTAyMDAiLCJkZXNjcmlwdGlvbiI6IiIsIm5hbWUiOiIzcmQgcGFydHkifSwiY3VzdG9tZmllbGRfMTQzMjgiOm51bGwsImF0dGFjaG1lbnQiOlt7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2F0dGFjaG1lbnQvMTUzNzYyIiwiaWQiOiIxNTM3NjIiLCJmaWxlbmFtZSI6InRlc3p0LnR4dCIsImF1dGhvciI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi91c2VyP3VzZXJuYW1lPWtpc3NiYWwiLCJuYW1lIjoia2lzc2JhbCIsImtleSI6IkpJUkFVU0VSMTk1NDciLCJlbWFpbEFkZHJlc3MiOiJLaXNzLkJhbGF6c0BhZWdvbi5odSIsImF2YXRhclVybHMiOnsiNDh4NDgiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9vd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIyNHgyNCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9c21hbGwmb3duZXJJZD1KSVJBVVNFUjE5NTQ3JmF2YXRhcklkPTE0OTA2IiwiMTZ4MTYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9zZWN1cmUvdXNlcmF2YXRhcj9zaXplPXhzbWFsbCZvd25lcklkPUpJUkFVU0VSMTk1NDcmYXZhdGFySWQ9MTQ5MDYiLCIzMngzMiI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS91c2VyYXZhdGFyP3NpemU9bWVkaXVtJm93bmVySWQ9SklSQVVTRVIxOTU0NyZhdmF0YXJJZD0xNDkwNiJ9LCJkaXNwbGF5TmFtZSI6Iktpc3MsIEJhbMOhenMiLCJhY3RpdmUiOnRydWUsInRpbWVab25lIjoiRXVyb3BlL0J1ZGFwZXN0In0sImNyZWF0ZWQiOiIyMDIzLTA2LTI2VDEzOjIyOjU3Ljc1NSswMjAwIiwic2l6ZSI6NywibWltZVR5cGUiOiJ0ZXh0L3BsYWluIiwiY29udGVudCI6Imh0dHBzOi8vamlyYS10ZXN0LmFlZ29uLmh1L3NlY3VyZS9hdHRhY2htZW50LzE1Mzc2Mi90ZXN6dC50eHQifV0sImN1c3RvbWZpZWxkXzE0NDQwIjpudWxsLCJjdXN0b21maWVsZF8xNDQ0MyI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMjIiOm51bGwsImN1c3RvbWZpZWxkXzE0NDQ0IjpudWxsLCJjdXN0b21maWVsZF8xNDMyMyI6bnVsbCwiY3VzdG9tZmllbGRfMTQyMDAiOm51bGwsImN1c3RvbWZpZWxkXzE0NDQyIjpudWxsLCJjdXN0b21maWVsZF8xNDMyMSI6eyJzZWxmIjoiaHR0cHM6Ly9qaXJhLXRlc3QuYWVnb24uaHUvcmVzdC9hcGkvMi9jdXN0b21GaWVsZE9wdGlvbi8xNzM3OCIsInZhbHVlIjoiQWxhY3NvbnkgLSAgS2lzZWJiIGhpYmEgKHBsIGvDqW55ZWxtaSBmdW5rY2nDsykgYXogYWxrYWxtYXrDoXMgbcWxa8O2ZMOpc8OpYmVuIiwiaWQiOiIxNzM3OCIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE0NDQ3IjpudWxsLCJjdXN0b21maWVsZF8xNDMyNiI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NDgiOm51bGwsImN1c3RvbWZpZWxkXzE0MzI3IjpudWxsLCJjdXN0b21maWVsZF8xNDMyNCI6bnVsbCwiY3VzdG9tZmllbGRfMTQ0NDYiOm51bGwsImN1c3RvbWZpZWxkXzE0MzI1Ijp7InNlbGYiOiJodHRwczovL2ppcmEtdGVzdC5hZWdvbi5odS9yZXN0L2FwaS8yL2N1c3RvbUZpZWxkT3B0aW9uLzE3MzcwIiwidmFsdWUiOiJBbGFjc29ueSAtIEVneWVkaSBmZWxoYXN6bsOhbMOzaSBlc2V0IiwiaWQiOiIxNzM3MCIsImRpc2FibGVkIjpmYWxzZX0sImN1c3RvbWZpZWxkXzE0NDM4IjpudWxsLCJjdXN0b21maWVsZF8xNDQzOSI6bnVsbCwiY3VzdG9tZmllbGRfMTQzMTgiOm51bGx9fQ==`),
			MantisID: "15279",
		},
	} {
		var issue JIRAIssue
		if err := json.NewDecoder(strings.NewReader(tC.Src)).Decode(&issue); err != nil {
			t.Fatal(err)
		}
		b, err := issue.Fields.MarshalJSON()
		t.Log(string(b))
		if err != nil {
			t.Fatal(err)
		}
		t.Log(issue)
		if issue.Fields.MantisID != tC.MantisID {
			t.Errorf("mantisID: got %q, wanted %q", issue.Fields.MantisID, tC.MantisID)

		}
	}

}

func b64decode(s string) string {
	var buf strings.Builder
	if _, err := io.Copy(&buf, base64.NewDecoder(base64.StdEncoding, strings.NewReader(s))); err != nil {
		panic(err)
	}
	return buf.String()
}