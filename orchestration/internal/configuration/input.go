package configuration

import (
	"io/ioutil"
	"errors"
	"regexp"
	"strings"
	"gopkg.in/yaml.v2"
	"../util"
)

const NAME_MIN_LENGTH=4

var defaultConfig string

type InputVars struct {
	Debug bool `yaml:"Debug"`

	ProjectName string `yaml:"ProjectName"`
	ProjectId string `yaml:"ProjectId"`

	Domain string `yaml:"Domain"`
	ClusterId string `yaml:"ClusterId"`

	AggregatedLogging bool `yaml:"AggregatedLogging"`
	ClusterMetrics bool `yaml:"ClusterMetrics"`

	Storage StorageConfig `yaml:"Storage"`
	NodeCounts NodeCountConfig `yaml:"NodeCounts"`

	NodeTypes NodeTypeConfig `yaml:"NodeTypes"`

	RegistryToS3 bool `yaml:"RegistryToS3"`

	AwsConfig AwsConfiguration `yaml:"AwsConfig"`
}

type StorageConfig struct {
	EnableEfs bool `yaml:"EnableEfs"`
	EncryptEfs bool `yaml:"EncryptEfs"`

	EnableEbs bool `yaml:"EnableEbs"`
	EncryptEbs bool `yaml:"EncryptEbs"`

	Default string `yaml:"Default"`
}

type NodeCountConfig struct {
	Master int `yaml:"Master"`
	Infra int `yaml:"Infra"`
	App int `yaml:"App"`
}

type NodeTypeConfig struct {
	Bastion string `yaml:"Bastion"`
	Master string `yaml:"Master"`
	Infra string `yaml:"Infra"`
	App string `yaml:"App"`
}

type AwsConfiguration struct {
	Region string `yaml:"Region"`
	KeyId string `yaml:"KeyId"`
	SecretKey string `yaml:"SecretKey"`
}


func init() {
	defaultConfig = "config.default.yaml"
}

func LoadConfigFromFile(filename string) *InputVars {
	content, err := ioutil.ReadFile(filename)
	util.ExitOnError("Cannot Open configuration file", err)

	return ParseInputVars(content)
}

func ParseInputVars (config []byte) *InputVars {
	vars := InputVars{}
	yaml.Unmarshal(config, &vars)

	if len(vars.ProjectName) >= NAME_MIN_LENGTH && len(vars.ProjectId) < NAME_MIN_LENGTH {
		vars.ProjectId = util.EncodeProjectId(vars.ProjectName)
	}

	vars.Storage.Default = strings.ToLower(vars.Storage.Default)

	return &vars
}

func DefaultConfig() *InputVars {
	config := LoadConfigFromFile(defaultConfig)
	return config
}

func (vars *InputVars) Validate() error {
	if len(vars.ProjectName) < NAME_MIN_LENGTH {
		return errors.New("invalid argument: Your Project name should contain at least 4 characters")
	}

	if len(vars.ProjectId) < NAME_MIN_LENGTH {
		return errors.New("invalid argument: Your Project ID should contain at least 4 characters")
	}

	if vars.NodeCounts.Master < 1 {
		return errors.New("invalid argument: Master Node Count cannot be 0")
	}

	if vars.NodeCounts.Master == 2 {
		return errors.New("invalid argument: Master Node Count should be 1 or 3 or more")
	}

	if vars.NodeCounts.Infra < 1 {
		return errors.New("invalid argument: Infrastructure Node Count cannot be 0")
	}

	if vars.NodeCounts.App < 1 {
		return errors.New("invalid argument: Application Node Count cannot be 0")
	}

	if !vars.Storage.EnableEfs && !vars.Storage.EnableEbs {
		return errors.New("invalid storage config: There should be at least one enabled persistence provider")
	}

	if vars.Storage.Default != "" && vars.Storage.Default != "ebs" && vars.Storage.Default != "efs" {
		return errors.New("invalid argument: Invalid default storage provider")
	}

	if vars.Storage.Default == "ebs" && !vars.Storage.EnableEbs {
		return errors.New("invalid storage config: The chosen default storage class is not enabled")
	}

	if vars.Storage.Default == "efs" && !vars.Storage.EnableEfs {
		return errors.New("invalid storage config: The chosen default storage class is not enabled")
	}

	if vars.AggregatedLogging && !vars.Storage.EnableEbs {
		return errors.New("invalid storage config: Aggregated logging depends on EBS storage")
	}

	if len(vars.ClusterId) < 1 {
		return errors.New("invalid cluster Id")
	}

	// @todo validate instance types more precise
	r := regexp.MustCompile("^([tmcpxridgfh][0-9]|x[0-9]e|m[0-9]d)\\.[\\w]+$")
	if !r.MatchString(vars.NodeTypes.Bastion) {
		return errors.New("invalid argument: Invalid Bastion type (" + vars.NodeTypes.Bastion + ")")
	}

	if !r.MatchString(vars.NodeTypes.Master) {
		return errors.New("invalid argument: Invalid Master type (" + vars.NodeTypes.Master + ")")
	}

	if !r.MatchString(vars.NodeTypes.Infra) {
		return errors.New("invalid argument: Invalid Infrastructure type (" + vars.NodeTypes.Infra + ")")
	}

	if !r.MatchString(vars.NodeTypes.App) {
		return errors.New("invalid argument: Invalid Application type (" + vars.NodeTypes.App + ")")
	}

	r = regexp.MustCompile("^([a-zA-Z0-9-_]+\\.)*[a-zA-Z0-9][a-zA-Z0-9-_]+\\.[a-zA-Z]{2,11}$")
	vars.Domain = strings.ToLower(vars.Domain)
	if !r.MatchString(vars.Domain) {
		return errors.New("invalid argument: Invalid Domain given (" + vars.Domain + ")")
	}

	r = regexp.MustCompile("^[a-z]{2}-[a-z]{4,}-[\\d]$")
	vars.AwsConfig.Region = strings.ToLower(vars.AwsConfig.Region)
	if !r.MatchString(vars.AwsConfig.Region) {
		return errors.New("invalid argument: Invalid AWS region (" + vars.AwsConfig.Region + ")")
	}

	return nil
}

func (vars *InputVars) MergeCmdFlags(flags CmdFlags) {
	vars.Debug = vars.Debug || flags.Debug

	if len(flags.ProjectName) > NAME_MIN_LENGTH {
		vars.ProjectName = flags.ProjectName
	}

	if len(flags.ProjectId) > NAME_MIN_LENGTH {
		vars.ProjectId = flags.ProjectId
	}

	if len(flags.AwsConfig.Region) > 0 {
		vars.AwsConfig.Region = flags.AwsConfig.Region
	}

	if len(flags.AwsConfig.KeyId) > 0 && len(flags.AwsConfig.SecretKey) > 0 {
		vars.AwsConfig.KeyId = flags.AwsConfig.KeyId
		vars.AwsConfig.SecretKey = flags.AwsConfig.SecretKey
	}
}