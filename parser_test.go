package nginxparser

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"
	"text/template"
)

const (
	fixtureTemplate = `
{{define "Directives"}} []*Directive{
	{{- range $directive := . }}
	{
		Line:      {{ $directive.Line }},
		FileName:  {{ quote $directive.FileName }},
		Directive: {{ quote $directive.Directive }},
{{ if ne (len $directive.Args) 0 }} Args: []string{ {{- range $arg := $directive.Args }} {{ quote $arg }}, {{- end }} }, {{ end }} 
{{ if ne (len $directive.Block) 0 }} Block: {{ template "Directives" $directive.Block }} {{ end }} 
{{ if ne (len $directive.Comment) 0 }} Comment: {{ quote $directive.Comment }},  {{ end }} 
	},
	{{- end }}
}, {{end}}
{{ template "Directives" . }}
`
)

func buildFixture(directives []*Directive) (string, error) {
	var buf bytes.Buffer
	err := template.Must(template.New("fixtureTemplate").Funcs(map[string]interface{}{
		"quote": strconv.Quote,
	}).Parse(fixtureTemplate)).Execute(&buf, directives)
	return buf.String(), err
}

type ParseFixture struct {
	name       string
	options    *ParseOptions
	directives []*Directive
}

func TestParse(t *testing.T) {
	parseFixtures := []*ParseFixture{
		{
			name: "bad-args",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/bad-args/nginx.conf",
					Directive: "user",
				},
				{
					Line:      2,
					FileName:  "testdata/bad-args/nginx.conf",
					Directive: "events",
				},
				{
					Line:      3,
					FileName:  "testdata/bad-args/nginx.conf",
					Directive: "http",
				},
			},
		},
		{
			name: "comments-between-args",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/comments-between-args/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      1,
							FileName:  "testdata/comments-between-args/nginx.conf",
							Directive: "#",
							Comment:   "comment 1",
						},
						{
							Line:      2,
							FileName:  "testdata/comments-between-args/nginx.conf",
							Directive: "log_format",
							Args:      []string{"#arg 1", "#arg 2"},
							Comment:   "comment 2 comment 3 comment 4 comment 5",
						},
					},
				},
			},
		},
		{
			name: "directive-with-space",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/directive-with-space/nginx.conf",
					Directive: "events",
				},
				{
					Line:      3,
					FileName:  "testdata/directive-with-space/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      4,
							FileName:  "testdata/directive-with-space/nginx.conf",
							Directive: "map",
							Args:      []string{"$http_user_agent", "$mobile"},
							Block: []*Directive{
								{
									Line:      5,
									FileName:  "testdata/directive-with-space/nginx.conf",
									Directive: "default",
									Args:      []string{"0"},
								},
								{
									Line:      6,
									FileName:  "testdata/directive-with-space/nginx.conf",
									Directive: "~Opera Mini",
									Args:      []string{"1"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "empty-value-map",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/empty-value-map/nginx.conf",
					Directive: "events",
				},
				{
					Line:      3,
					FileName:  "testdata/empty-value-map/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      4,
							FileName:  "testdata/empty-value-map/nginx.conf",
							Directive: "map",
							Args:      []string{"string", "$variable"},
							Block: []*Directive{
								{
									Line:      5,
									FileName:  "testdata/empty-value-map/nginx.conf",
									Directive: "",
									Args:      []string{"$arg"},
								},
								{
									Line:      6,
									FileName:  "testdata/empty-value-map/nginx.conf",
									Directive: "*.example.com",
									Args:      []string{""},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "includes-globbed",
			options: &ParseOptions{
				Root: filepath.Join("testdata", "includes-globbed"),
			},
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/includes-globbed/nginx.conf",
					Directive: "events",
				},
				{
					Line:      2,
					FileName:  "testdata/includes-globbed/nginx.conf",
					Directive: "include",
					Args:      []string{"http.conf"},
					Block: []*Directive{
						{
							Line:      1,
							FileName:  "testdata/includes-globbed/http.conf",
							Directive: "http",
							Block: []*Directive{
								{
									Line:      2,
									FileName:  "testdata/includes-globbed/http.conf",
									Directive: "include",
									Args:      []string{"servers/*.conf"},
									Block: []*Directive{
										{
											Line:      1,
											FileName:  "testdata/includes-globbed/servers/server1.conf",
											Directive: "server",
											Block: []*Directive{
												{
													Line:      2,
													FileName:  "testdata/includes-globbed/servers/server1.conf",
													Directive: "listen",
													Args:      []string{"8080"},
												},
												{
													Line:      3,
													FileName:  "testdata/includes-globbed/servers/server1.conf",
													Directive: "include",
													Args:      []string{"locations/*.conf"},
													Block: []*Directive{
														{
															Line:      1,
															FileName:  "testdata/includes-globbed/locations/location1.conf",
															Directive: "location",
															Args:      []string{"/foo"},
															Block: []*Directive{
																{
																	Line:      2,
																	FileName:  "testdata/includes-globbed/locations/location1.conf",
																	Directive: "return",
																	Args:      []string{"200", "foo"},
																},
															},
														},
														{
															Line:      1,
															FileName:  "testdata/includes-globbed/locations/location2.conf",
															Directive: "location",
															Args:      []string{"/bar"},
															Block: []*Directive{
																{
																	Line:      2,
																	FileName:  "testdata/includes-globbed/locations/location2.conf",
																	Directive: "return",
																	Args:      []string{"200", "bar"},
																},
															},
														},
													},
												},
											},
										},
										{
											Line:      1,
											FileName:  "testdata/includes-globbed/servers/server2.conf",
											Directive: "server",
											Block: []*Directive{
												{
													Line:      2,
													FileName:  "testdata/includes-globbed/servers/server2.conf",
													Directive: "listen",
													Args:      []string{"8081"},
												},
												{
													Line:      3,
													FileName:  "testdata/includes-globbed/servers/server2.conf",
													Directive: "include",
													Args:      []string{"locations/*.conf"},
													Block: []*Directive{
														{
															Line:      1,
															FileName:  "testdata/includes-globbed/locations/location1.conf",
															Directive: "location",
															Args:      []string{"/foo"},
															Block: []*Directive{
																{
																	Line:      2,
																	FileName:  "testdata/includes-globbed/locations/location1.conf",
																	Directive: "return",
																	Args:      []string{"200", "foo"},
																},
															},
														},
														{
															Line:      1,
															FileName:  "testdata/includes-globbed/locations/location2.conf",
															Directive: "location",
															Args:      []string{"/bar"},
															Block: []*Directive{
																{
																	Line:      2,
																	FileName:  "testdata/includes-globbed/locations/location2.conf",
																	Directive: "return",
																	Args:      []string{"200", "bar"},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "includes-regular",
			options: &ParseOptions{
				Root: filepath.Join("testdata", "includes-regular"),
			},
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/includes-regular/nginx.conf",
					Directive: "events",
				},
				{
					Line:      2,
					FileName:  "testdata/includes-regular/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      3,
							FileName:  "testdata/includes-regular/nginx.conf",
							Directive: "include",
							Args:      []string{"conf.d/server.conf"},
							Block: []*Directive{
								{
									Line:      1,
									FileName:  "testdata/includes-regular/conf.d/server.conf",
									Directive: "server",

									Block: []*Directive{
										{
											Line:      2,
											FileName:  "testdata/includes-regular/conf.d/server.conf",
											Directive: "listen",
											Args:      []string{"127.0.0.1:8080"},
										},
										{
											Line:      3,
											FileName:  "testdata/includes-regular/conf.d/server.conf",
											Directive: "server_name",
											Args:      []string{"default_server"},
										},
										{
											Line:      4,
											FileName:  "testdata/includes-regular/conf.d/server.conf",
											Directive: "include",
											Args:      []string{"foo.conf"},
											Block: []*Directive{
												{
													Line:      1,
													FileName:  "testdata/includes-regular/foo.conf",
													Directive: "location",
													Args:      []string{"/foo"},
													Block: []*Directive{
														{
															Line:      2,
															FileName:  "testdata/includes-regular/foo.conf",
															Directive: "return",
															Args:      []string{"200", "foo"},
														},
													},
												},
											},
										},
										{
											Line:      5,
											FileName:  "testdata/includes-regular/conf.d/server.conf",
											Directive: "include",
											Args:      []string{"bar.conf"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "lua-block-larger",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/lua-block-larger/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/lua-block-larger/nginx.conf",
							Directive: "content_by_lua_block",
							Args:      []string{"\n        ngx.req.read_body()  -- explicitly read the req body\n        local data = ngx.req.get_body_data()\n        if data then\n            ngx.say(\"body data:\")\n            ngx.print(data)\n            return\n        end\n\n        -- body may get buffered in a temp file:\n        local file = ngx.req.get_body_file()\n        if file then\n            ngx.say(\"body is in file \", file)\n        else\n            ngx.say(\"no body found\")\n        end"},
						},
						{
							Line:      19,
							FileName:  "testdata/lua-block-larger/nginx.conf",
							Directive: "access_by_lua_block",
							Args:      []string{"\n        -- check the client IP address is in our black list\n        if ngx.var.remote_addr == \"132.5.72.3\" then\n            ngx.exit(ngx.HTTP_FORBIDDEN)\n        end\n\n        -- check if the URI contains bad words\n        if ngx.var.uri and\n               string.match(ngx.var.request_body, \"evil\")\n        then\n            return ngx.redirect(\"/terms_of_use.html\")\n        end\n\n        -- tests passed"},
						},
					},
				},
			},
		},
		{
			name: "lua-block-simple",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/lua-block-simple/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "init_by_lua_block",
							Args:      []string{"\n        print(\"Lua block code with curly brace str {\")"},
						},
						{
							Line:      5,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "init_worker_by_lua_block",
							Args:      []string{"\n        print(\"Work that every worker\")"},
						},
						{
							Line:      8,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "body_filter_by_lua_block",
							Args:      []string{"\n        local data, eof = ngx.arg[1], ngx.arg[2]"},
						},
						{
							Line:      11,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "header_filter_by_lua_block",
							Args:      []string{"\n        ngx.header[\"content-length\"] = nil"},
						},
						{
							Line:      14,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      15,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
								},
								{
									Line:      16,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "location",
									Args:      []string{"/"},
									Block: []*Directive{
										{
											Line:      17,
											FileName:  "testdata/lua-block-simple/nginx.conf",
											Directive: "content_by_lua_block",
											Args:      []string{"\n                ngx.say(\"I need no extra escaping here, for example: \\r\\nblah\")"},
										},
										{
											Line:      20,
											FileName:  "testdata/lua-block-simple/nginx.conf",
											Directive: "return",
											Args:      []string{"200", "foo bar baz"},
										},
									},
								},
								{
									Line:      22,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "ssl_certificate_by_lua_block",
									Args:      []string{"\n            print(\"About to initiate a new SSL handshake!\")"},
								},
								{
									Line:      25,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "location",
									Args:      []string{"/a"},
									Block: []*Directive{
										{
											Line:      26,
											FileName:  "testdata/lua-block-simple/nginx.conf",
											Directive: "client_max_body_size",
											Args:      []string{"100k"},
										},
										{
											Line:      27,
											FileName:  "testdata/lua-block-simple/nginx.conf",
											Directive: "client_body_buffer_size",
											Args:      []string{"100k"},
										},
									},
								},
							},
						},
						{
							Line:      31,
							FileName:  "testdata/lua-block-simple/nginx.conf",
							Directive: "upstream",
							Args:      []string{"foo"},
							Block: []*Directive{
								{
									Line:      32,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "server",
									Args:      []string{"127.0.0.1"},
								},
								{
									Line:      33,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "balancer_by_lua_block",
									Args:      []string{"\n            -- use Lua that'll do something interesting here with external bracket for testing {"},
								},
								{
									Line:      36,
									FileName:  "testdata/lua-block-simple/nginx.conf",
									Directive: "log_by_lua_block",
									Args:      []string{"\n            print(\"I need no extra escaping here, for example: \\r\\nblah\")"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "lua-block-tricky",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/lua-block-tricky/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/lua-block-tricky/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      3,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
								},
								{
									Line:      4,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "server_name",
									Args:      []string{"content_by_lua_block"},
								},
								{
									Line:      4,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "#",
									Comment:   " make sure this doesn't trip up lexers",
								},
								{
									Line:      5,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "set_by_lua_block",
									Args:      []string{"$res", " -- irregular lua block directive\n            local a = 32\n            local b = 56\n\n            ngx.var.diff = a - b;  -- write to $diff directly\n            return a + b;          -- return the $sum value normally"},
								},
								{
									Line:      12,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "rewrite_by_lua_block",
									Args:      []string{" -- have valid braces in Lua code and quotes around directive\n            do_something(\"hello, world!\\nhiya\\n\")\n            a = { 1, 2, 3 }\n            btn = iup.button({title=\"ok\"})"},
								},
							},
						},
						{
							Line:      18,
							FileName:  "testdata/lua-block-tricky/nginx.conf",
							Directive: "upstream",
							Args:      []string{"content_by_lua_block"},
							Block: []*Directive{
								{
									Line:      19,
									FileName:  "testdata/lua-block-tricky/nginx.conf",
									Directive: "#",
									Comment:   " stuff",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "messy",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/messy/nginx.conf",
					Directive: "user",
					Args:      []string{"nobody"},
				},
				{
					Line:      2,
					FileName:  "testdata/messy/nginx.conf",
					Directive: "#",
					Comment:   " hello\\n\\\\n\\\\\\n worlddd  \\#\\\\#\\\\\\# dfsf\\n \\\\n \\\\\\n \\",
				},
				{
					Line:      3,
					FileName:  "testdata/messy/nginx.conf",
					Directive: "events",

					Block: []*Directive{
						{
							Line:      3,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "worker_connections",
							Args:      []string{"2048"},
						},
					},
				},
				{
					Line:      5,
					FileName:  "testdata/messy/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      5,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "#",
							Comment:   "forteen",
						},
						{
							Line:      6,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "#",
							Comment:   " this is a comment",
						},
						{
							Line:      7,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "access_log",
							Args:      []string{"off"},
						},
						{
							Line:      7,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "default_type",
							Args:      []string{"text/plain"},
						},
						{
							Line:      7,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "error_log",
							Args:      []string{"off"},
						},
						{
							Line:      8,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      9,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "listen",
									Args:      []string{"8083"},
								},
								{
									Line:      10,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "return",
									Args:      []string{"200", "Ser\" ' ' ver\\ \\ $server_addr:\\$server_port\n\nTime: $time_local\n\n"},
								},
							},
						},
						{
							Line:      12,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      12,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "listen",
									Args:      []string{"8080"},
								},
								{
									Line:      13,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "root",
									Args:      []string{"/usr/share/nginx/html"},
								},
								{
									Line:      14,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"~", "/hello/world;"},
									Block: []*Directive{
										{
											Line:      14,
											FileName:  "testdata/messy/nginx.conf",
											Directive: "return",
											Args:      []string{"301", "/status.html"},
										},
									},
								},
								{
									Line:      15,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/foo"},
								},
								{
									Line:      15,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/bar"},
								},
								{
									Line:      16,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/{;} # ab"},
								},
								{
									Line:      16,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "#",
									Comment:   " hello",
								},
								{
									Line:      17,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "if",
									Args:      []string{"$request_method", "=", "P{O)###;ST"},
								},
								{
									Line:      18,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/status.html"},
									Block: []*Directive{
										{
											Line:      19,
											FileName:  "testdata/messy/nginx.conf",
											Directive: "try_files",
											Args:      []string{"/abc/${uri}", "/abc/${uri}.html", "=404"},
										},
									},
								},
								{
									Line:      21,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/sta;\n                    tus"},
									Block: []*Directive{
										{
											Line:      22,
											FileName:  "testdata/messy/nginx.conf",
											Directive: "return",
											Args:      []string{"302", "/status.html"},
										},
									},
								},
								{
									Line:      23,
									FileName:  "testdata/messy/nginx.conf",
									Directive: "location",
									Args:      []string{"/upstream_conf"},
									Block: []*Directive{
										{
											Line:      23,
											FileName:  "testdata/messy/nginx.conf",
											Directive: "return",
											Args:      []string{"200", "/status.html"},
										},
									},
								},
							},
						},
						{
							Line:      24,
							FileName:  "testdata/messy/nginx.conf",
							Directive: "server",
						},
					},
				},
			},
		},
		{
			name: "missing-semicolon-above",
		},
		{
			name: "missing-semicolon-below",
		},
		{
			name: "quote-behavior",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/quote-behavior/nginx.conf",
					Directive: "outer-quote",
					Args:      []string{"left", "-quote", "right-\"quote\"", "inner\"-\"quote"},
				},
				{
					Line:      2,
					FileName:  "testdata/quote-behavior/nginx.conf",
					Directive: "",
					Args:      []string{"", "left-empty", "right-empty\"\"", "inner\"\"empty", "right-empty-single\""},
				},
			},
		},
		{
			name: "quoted-right-brace",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/quoted-right-brace/nginx.conf",
					Directive: "events",
				},
				{
					Line:      2,
					FileName:  "testdata/quoted-right-brace/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      3,
							FileName:  "testdata/quoted-right-brace/nginx.conf",
							Directive: "log_format",
							Args:      []string{"main", "escape=json", "{ \"@timestamp\": \"$time_iso8601\", \"server_name\": \"$server_name\", \"host\": \"$host\", \"status\": \"$status\", \"request\": \"$request\", \"uri\": \"$uri\", \"args\": \"$args\", \"https\": \"$https\", \"request_method\": \"$request_method\", \"referer\": \"$http_referer\", \"agent\": \"$http_user_agent\"}"},
						},
					},
				},
			},
		},
		{
			name: "russian-text",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/russian-text/nginx.conf",
					Directive: "env",
					Args:      []string{"русский текст"},
				},
				{
					Line:      2,
					FileName:  "testdata/russian-text/nginx.conf",
					Directive: "events",
				},
			},
		},
		{
			name: "simple",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/simple/nginx.conf",
					Directive: "events",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/simple/nginx.conf",
							Directive: "worker_connections",
							Args:      []string{"1024"},
						},
					},
				},
				{
					Line:      5,
					FileName:  "testdata/simple/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      6,
							FileName:  "testdata/simple/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      7,
									FileName:  "testdata/simple/nginx.conf",
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
								},
								{
									Line:      8,
									FileName:  "testdata/simple/nginx.conf",
									Directive: "server_name",
									Args:      []string{"default_server"},
								},
								{
									Line:      9,
									FileName:  "testdata/simple/nginx.conf",
									Directive: "location",
									Args:      []string{"/"},
									Block: []*Directive{
										{
											Line:      10,
											FileName:  "testdata/simple/nginx.conf",
											Directive: "return",
											Args:      []string{"200", "foo bar baz"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "simple-with-if",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/simple-with-if/nginx.conf",
					Directive: "events",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/simple-with-if/nginx.conf",
							Directive: "worker_connections",
							Args:      []string{"1024"},
						},
					},
				},
				{
					Line:      5,
					FileName:  "testdata/simple-with-if/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      6,
							FileName:  "testdata/simple-with-if/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      7,
									FileName:  "testdata/simple-with-if/nginx.conf",
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
								},
								{
									Line:      8,
									FileName:  "testdata/simple-with-if/nginx.conf",
									Directive: "server_name",
									Args:      []string{"default_server"},
								},
								{
									Line:      10,
									FileName:  "testdata/simple-with-if/nginx.conf",
									Directive: "location",
									Args:      []string{"/"},
									Block: []*Directive{
										{
											Line:      11,
											FileName:  "testdata/simple-with-if/nginx.conf",
											Directive: "if",
											Args:      []string{"$scheme", "=", "http"},
											Block: []*Directive{
												{
													Line:      12,
													FileName:  "testdata/simple-with-if/nginx.conf",
													Directive: "return",
													Args:      []string{"200", "foo bar"},
												},
											},
										},
										{
											Line:      14,
											FileName:  "testdata/simple-with-if/nginx.conf",
											Directive: "return",
											Args:      []string{"200", "foo bar baz"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "spelling-mistake",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/spelling-mistake/nginx.conf",
					Directive: "events",
				},
				{
					Line:      3,
					FileName:  "testdata/spelling-mistake/nginx.conf",
					Directive: "http",

					Block: []*Directive{
						{
							Line:      4,
							FileName:  "testdata/spelling-mistake/nginx.conf",
							Directive: "server",

							Block: []*Directive{
								{
									Line:      5,
									FileName:  "testdata/spelling-mistake/nginx.conf",
									Directive: "location",
									Args:      []string{"/"},
									Block: []*Directive{
										{
											Line:      6,
											FileName:  "testdata/spelling-mistake/nginx.conf",
											Directive: "#",

											Comment: "directive is misspelled",
										},
										{
											Line:      7,
											FileName:  "testdata/spelling-mistake/nginx.conf",
											Directive: "proxy_passs",
											Args:      []string{"http://foo.bar"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "with-comments",
			directives: []*Directive{
				{
					Line:      1,
					FileName:  "testdata/with-comments/nginx.conf",
					Directive: "events",
					Block: []*Directive{
						{
							Line:      2,
							FileName:  "testdata/with-comments/nginx.conf",
							Directive: "worker_connections",
							Args:      []string{"1024"},
						},
					},
				},
				{
					Line:      4,
					FileName:  "testdata/with-comments/nginx.conf",
					Directive: "#",
					Comment:   "comment",
				},
				{
					Line:      5,
					FileName:  "testdata/with-comments/nginx.conf",
					Directive: "http",
					Block: []*Directive{
						{
							Line:      6,
							FileName:  "testdata/with-comments/nginx.conf",
							Directive: "server",
							Block: []*Directive{
								{
									Line:      7,
									FileName:  "testdata/with-comments/nginx.conf",
									Directive: "listen",
									Args:      []string{"127.0.0.1:8080"},
								},
								{
									Line:      7,
									FileName:  "testdata/with-comments/nginx.conf",
									Directive: "#",
									Comment:   "listen",
								},
								{
									Line:      8,
									FileName:  "testdata/with-comments/nginx.conf",
									Directive: "server_name",
									Args:      []string{"default_server"},
								},
								{
									Line:      9,
									FileName:  "testdata/with-comments/nginx.conf",
									Directive: "location",
									Args:      []string{"/"},
									Block: []*Directive{
										{
											Line:      9,
											FileName:  "testdata/with-comments/nginx.conf",
											Directive: "#",

											Comment: "# this is brace",
										},
										{
											Line:      10,
											FileName:  "testdata/with-comments/nginx.conf",
											Directive: "#",
											Comment:   " location /",
										},
										{
											Line:      11,
											FileName:  "testdata/with-comments/nginx.conf",
											Directive: "return",
											Args:      []string{"200", "foo bar baz"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, fixture := range parseFixtures {
		t.Run(fixture.name, func(t *testing.T) {
			parser := New(fixture.options)
			payload, err := parser.ParseFile(filepath.Join("testdata", fixture.name, "nginx.conf"))
			if fixture.directives == nil {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error %s", err)
				}
				b1, _ := json.Marshal(fixture.directives)
				b2, _ := json.Marshal(payload)
				if string(b1) != string(b2) {
					f1, _ := buildFixture(payload)
					t.Fatalf("expected: %s\nbut got: %s, ref %s", b1, b2, f1)
				}
			}
		})
	}
}
