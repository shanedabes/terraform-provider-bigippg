/*
Original work from https://github.com/DealerDotCom/terraform-provider-bigip
Modifications Copyright 2019 F5 Networks Inc.
This Source Code Form is subject to the terms of the Mozilla Public License, v. 2.0.
If a copy of the MPL was not distributed with this file,You can obtain one at https://mozilla.org/MPL/2.0/.
*/
package bigippg

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const DEFAULT_PARTITION = "Common"

func Provider() terraform.ResourceProvider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Domain name/IP of the BigIP",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_HOST", nil),
			},
			"port": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Management Port to connect to Bigip",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_PORT", nil),
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Username with API access to the BigIP",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_USER", nil),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The user's password",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_PASSWORD", nil),
			},
			"token_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable to use an external authentication source (LDAP, TACACS, etc)",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_TOKEN_AUTH", nil),
			},
			"teem_disable": {
				Type:     schema.TypeBool,
				Optional: true,
				//Default:     false,
				Description: "If this flag set to true,sending telemetry data to TEEM will be disabled",
				DefaultFunc: schema.EnvDefaultFunc("TEEM_DISABLE", false),
			},
			"login_ref": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "tmos",
				Description: "Login reference for token authentication (see BIG-IP REST docs for details)",
				DefaultFunc: schema.EnvDefaultFunc("BIGIP_LOGIN_REF", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"bigippg_ltm_monitor": resourceBigipLtmMonitor(),
		},
	}
	p.ConfigureFunc = func(d *schema.ResourceData) (interface{}, error) {
		terraformVersion := p.TerraformVersion
		if terraformVersion == "" {
			// Terraform 0.12 introduced this field to the protocol
			// We can therefore assume that if it's missing it's 0.10 or 0.11
			terraformVersion = "0.11+compatible"
		}
		return providerConfigure(d, terraformVersion)
	}
	return p
}

func providerConfigure(d *schema.ResourceData, terraformVersion string) (interface{}, error) {
	config := Config{
		Address:  d.Get("address").(string),
		Port:     d.Get("port").(string),
		Username: d.Get("username").(string),
		Password: d.Get("password").(string),
	}
	if d.Get("token_auth").(bool) {
		config.LoginReference = d.Get("login_ref").(string)
	}
	cfg, err := config.Client()
	if err != nil {
		return cfg, err
	}
	cfg.UserAgent = fmt.Sprintf("Terraform/%s", terraformVersion)
	cfg.UserAgent += fmt.Sprintf("/terraform-provider-bigip/%s", getVersion())
	//log.Printf("my app %s, commit %s, built at %s by %s", version, commit, date, builtBy)
	cfg.Teem = d.Get("teem_disable").(bool)
	return cfg, err
}

//Convert slice of strings to schema.TypeSet
func makeStringList(list *[]string) []interface{} {
	ilist := make([]interface{}, len(*list))
	for i, v := range *list {
		ilist[i] = v
	}
	return ilist
}

//Convert slice of strings to schema.Set
func makeStringSet(list *[]string) *schema.Set {
	ilist := make([]interface{}, len(*list))
	for i, v := range *list {
		ilist[i] = v
	}
	return schema.NewSet(schema.HashString, ilist)
}

//Convert schema.TypeList to a slice of strings
func listToStringSlice(s []interface{}) []string {
	list := make([]string, len(s))
	for i, v := range s {
		list[i] = v.(string)
	}
	return list
}

//Convert schema.Set to a slice of strings
func setToStringSlice(s *schema.Set) []string {
	list := make([]string, s.Len())
	for i, v := range s.List() {
		list[i] = v.(string)
	}
	return list
}

//Copy map values into an object where map key == object field name (e.g. map[foo] == &{Foo: ...}
func mapEntity(d map[string]interface{}, obj interface{}) {
	val := reflect.ValueOf(obj).Elem()
	for field := range d {
		f := val.FieldByName(strings.Title(field))
		if f.IsValid() {
			if f.Kind() == reflect.Slice {
				incoming := d[field].([]interface{})
				s := reflect.MakeSlice(f.Type(), len(incoming), len(incoming))
				for i := 0; i < len(incoming); i++ {
					s.Index(i).Set(reflect.ValueOf(incoming[i]))
				}
				f.Set(s)
			} else {
				f.Set(reflect.ValueOf(d[field]))
			}
		} else {
			f := val.FieldByName(strings.Title(toCamelCase(field)))
			f.Set(reflect.ValueOf(d[field]))
		}
	}
}

//Break a string in the format /Partition/name into a Partition / Name object
func parseF5Identifier(str string) (partition, name string) {
	if strings.HasPrefix(str, "/") {
		ary := strings.SplitN(strings.TrimPrefix(str, "/"), "/", 2)
		return ary[0], ary[1]
	}
	return "", str
}

// Convert Snakecase to Camelcase
func toCamelCase(str string) string {
	var link = regexp.MustCompile("(^[A-Za-z])|_([A-Za-z])")
	return link.ReplaceAllStringFunc(str, func(s string) string {
		return strings.ToUpper(strings.Replace(s, "_", "", -1))
	})
}

// Convert Camelcase to Snakecase
func toSnakeCase(str string) string {
	var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func getVersion() string {
	return ProviderVersion
}
