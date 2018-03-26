package ansible

import (
	"aws"
	"strings"
	"settings"
)

func GenerateOpenshiftInventory(filename string) *Inventory {
	masters := aws.MasterNodes()
	infra := aws.InfraNodes()
	apps := aws.AppNodes()
	bastion := aws.BastionNode()

	defaultSubdomain := infra[0].ExternalIp + ".xip.io"
	clusterHostname := masters[0].ExternalDns

	sshConfig := settings.NewSshConfig("ssh.cfg")
	bastionConfig := settings.NewHostConfig(bastion.ExternalDns)
	bastionConfig.AddVar("StrictHostKeyChecking", "no")
	bastionConfig.AddVar("ProxyCommand", "none")
	bastionConfig.AddVar("ForwardAgent", "yes")
	sshConfig.AddHost(bastionConfig)

	nodeConfig := settings.NewHostConfig("10.10.*")
	nodeConfig.AddVar("ProxyCommand", "ssh -W %h:%p centos@" + bastion.ExternalDns)
	nodeConfig.AddVar("StrictHostKeyChecking", "no")
	sshConfig.AddHost(nodeConfig)
	sshConfig.Write()

	inventory := NewInventory(filename)
	inventory.AddSection("OSEv3:children", []string{"masters", "nodes", "etcd"})

	var vars []string
	vars = append(vars, "ansible_user=centos", "ansible_become=true", "deployment_type=origin")
	vars = append(vars, "ansible_ssh_common_args='-F ssh.cfg'")
	vars = append(vars, "openshift_release=v3.7.1", "openshift_image_tag=v3.7.1")
	vars = append(vars, "openshift_router_selector='router=true'", "openshift_registry_selector='registry=true'")
	vars = append(vars, "openshift_master_default_subdomain=" + defaultSubdomain)
	vars = append(vars, "openshift_clock_enable=true", "openshift_use_dnsmasq=true", "os_firewall_use_firewalld=true")
	vars = append(vars, "openshift_master_cluster_hostname=" + clusterHostname, "openshift_master_cluster_public_hostname=" + clusterHostname)
	vars = append(vars, "openshift_disable_check=docker_storage,memory_availability,package_version", "openshift_enable_service_catalog=false")
	vars = append(vars, "openshift_master_identity_providers=[{'name': 'htpasswd_auth', 'login': 'true', 'challenge': 'true', 'kind': 'HTPasswdPasswordIdentityProvider', 'filename': '/etc/origin/master/htpasswd'}]")
	vars = append(vars, "openshift_master_htpasswd_users={'admin': '$apr1$zgSjCrLt$1KSuj66CggeWSv.D.BXOA1', 'user': '$apr1$.gw8w9i1$ln9bfTRiD6OwuNTG5LvW50'}")

	inventory.AddSection("OSEv3:vars", vars)

	nodesSection := generateNodeLines(masters, "openshift_schedulable=false", true)
	nodesSection = append(nodesSection, generateNodeLines(infra, "openshift_node_labels=\"{'router':'true','registry':'true'}\"", true)...)
	nodesSection = append(nodesSection, generateNodeLines(apps, "", false)...)


	inventory.AddSection("masters", generateNodeLines(masters, "", true))
	inventory.AddSection("etcd", extractNodeIps(masters, false))
	inventory.AddSection("nodes", nodesSection)

	return inventory
}

func generateNodeLines(nodes []aws.NodeInfo, extra string, public bool) []string {
	var lines []string

	for _,node := range nodes {
		lines = append(lines, generateNodeLine(node, extra, public))
	}

	return lines
}

func generateNodeLine(node aws.NodeInfo, extra string, public bool) string {
	var s string
	extra = strings.TrimSpace(extra)

	//s += node.ExternalIp
	s += node.InternalIp
	if extra != "" {
		s += " " + extra
	}

	s += " openshift_ip=" + node.InternalIp
	s += " openshift_hostname=" + node.InternalDns

	if public {
		s += " openshift_public_ip=" + node.ExternalIp
		s += " openshift_public_hostname=" + node.ExternalDns
	}

	return s
}

func extractNodeIps(nodes []aws.NodeInfo, public bool) []string {
	var ips []string

	for _,node := range nodes {
		if public {
			ips = append(ips, node.ExternalIp)
		} else {
			ips = append(ips, node.InternalIp)
		}
	}

	return ips
}