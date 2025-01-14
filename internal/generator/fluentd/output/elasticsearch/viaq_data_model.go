package elasticsearch

import (
	"fmt"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"

	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	. "github.com/openshift/cluster-logging-operator/internal/generator/fluentd/elements"
	corev1 "k8s.io/api/core/v1"
)

type Viaq struct {
	Elasticsearch *logging.Elasticsearch
}

const (
	AnnotationPrefix = "containerType.logging.openshift.io"
)

func ViaqDataModel(bufspec *logging.FluentdBufferSpec, secret *corev1.Secret, o logging.OutputSpec, op Options) []Element {

	modRecordDedot := RecordModifier{
		Records: []Record{
			{
				Key: "_dummy_",
				// Replace namespace label names that have '.' & '/' with '_'
				Expression: `${if m=record.dig("kubernetes","namespace_labels");record["kubernetes"]["namespace_labels"]={}.tap{|n|m.each{|k,v|n[k.gsub(/[.\/]/,'_')]=v}};end}`,
			},
			{
				Key: "_dummy2_",
				// Replace label names that have '.' & '/' with '_'
				Expression: `${if m=record.dig("kubernetes","labels");record["kubernetes"]["labels"]={}.tap{|n|m.each{|k,v|n[k.gsub(/[.\/]/,'_')]=v}};end}`,
			},
			{
				Key: "_dummy3_",
				// Replace flattened label names that have '.' & '/' with '_'
				Expression: `${if m=record.dig("kubernetes","flat_labels");record["kubernetes"]["flat_labels"]=[].tap{|n|m.each_with_index{|s, i|n[i] = s.gsub(/[.\/]/,'_')}};end}`,
			},
		},
		RemoveKeys: []string{"_dummy_, _dummy2_, _dummy3_"},
	}

	modRecordRebuildMessage := RecordModifier{
		Records: []Record{
			{
				Key:        "_dummy_",
				Expression: `${(require 'json';record['message']=JSON.dump(record['structured'])) if record['structured'] and record['viaq_index_name'] == 'app-write'}`,
			},
		},
		RemoveKeys: []string{"_dummy_"},
	}

	elements := []Element{
		Filter{
			Desc:      "dedot namespace_labels",
			MatchTags: "**",
			Element:   modRecordDedot,
		},
		Viaq{
			Elasticsearch: o.Elasticsearch,
		},
		Filter{
			Desc:      "rebuild message field if present",
			MatchTags: "**",
			Element:   modRecordRebuildMessage,
		},
	}

	if o.Elasticsearch == nil || (o.Elasticsearch.StructuredTypeKey == "" && o.Elasticsearch.StructuredTypeName == "" && !o.Elasticsearch.EnableStructuredContainerLogs) {
		recordModifier := RecordModifier{
			RemoveKeys: []string{KeyStructured},
		}
		if op[CharEncoding] != nil {
			recordModifier.CharEncoding = fmt.Sprintf("%v", op[CharEncoding])
		}
		elements = append(elements, Filter{
			Desc:      "remove structured field if present",
			MatchTags: "**",
			Element:   recordModifier,
		})
	}

	return elements
}

func (im Viaq) StructuredTypeKey() string {
	if im.Elasticsearch != nil && im.Elasticsearch.StructuredTypeKey != "" {
		return im.Elasticsearch.StructuredTypeKey
	}
	return ""
}
func (im Viaq) StructuredTypeName() string {
	if im.Elasticsearch != nil && im.Elasticsearch.StructuredTypeName != "" {
		return im.Elasticsearch.StructuredTypeName
	}
	return ""
}
func (im Viaq) StructuredTypeAnnotationPrefix() string {
	if im.Elasticsearch != nil && im.Elasticsearch.EnableStructuredContainerLogs {
		return AnnotationPrefix
	}
	return ""
}

func (im Viaq) Name() string {
	return "viaqDataIndexModel"
}

func (im Viaq) Template() string {
	return `{{define "viaqDataIndexModel" -}}
# Viaq Data Model
<filter **>
  @type viaq_data_model
  enable_openshift_model false
  enable_prune_empty_fields false
  rename_time false
  undefined_dot_replace_char UNUSED
  elasticsearch_index_prefix_field 'viaq_index_name'
  <elasticsearch_index_name>
    enabled 'true'
    tag "kubernetes.var.log.pods.openshift_** kubernetes.var.log.pods.openshift-*_** kubernetes.var.log.pods.default_** kubernetes.var.log.pods.kube-*_** var.log.pods.openshift_** var.log.pods.openshift-*_** var.log.pods.default_** var.log.pods.kube-*_** journal.system** system.var.log**"
    name_type static
    static_index_name infra-write
{{if (ne .StructuredTypeKey "") -}}
    structured_type_key {{ .StructuredTypeKey }}
{{ end -}}
{{if (ne .StructuredTypeName "") -}}
    structured_type_name {{ .StructuredTypeName }}
{{ end -}}
{{if (ne .StructuredTypeAnnotationPrefix "") -}}
    structured_type_annotation_prefix {{ .StructuredTypeAnnotationPrefix }}
{{ end -}}
  </elasticsearch_index_name>
  <elasticsearch_index_name>
    enabled 'true'
    tag "linux-audit.log** k8s-audit.log** openshift-audit.log** ovn-audit.log**"
    name_type static
    static_index_name audit-write
  </elasticsearch_index_name>
  <elasticsearch_index_name>
    enabled 'true'
    tag "**"
    name_type structured
    static_index_name app-write
{{if (ne .StructuredTypeKey "") -}}
    structured_type_key {{ .StructuredTypeKey }}
{{ end -}}
{{if (ne .StructuredTypeName "") -}}
    structured_type_name {{ .StructuredTypeName }}
{{ end -}}
{{if (ne .StructuredTypeAnnotationPrefix "") -}}
    structured_type_annotation_prefix {{ .StructuredTypeAnnotationPrefix }}
{{ end -}}
  </elasticsearch_index_name>
</filter>
<filter **>
  @type viaq_data_model
  enable_prune_labels true
  enable_openshift_model false
  rename_time false
  undefined_dot_replace_char UNUSED
  prune_labels_exclusions app_kubernetes_io_name,app_kubernetes_io_instance,app_kubernetes_io_version,app_kubernetes_io_component,app_kubernetes_io_part-of,app_kubernetes_io_managed-by,app_kubernetes_io_created-by
</filter>
{{end}}
`
}
