# compliance-operator

The compliance-operator is a OpenShift Operator that allows an administrator
to run compliance scans and provide remediations for the issues found. The
operator leverages OpenSCAP under the hood to perform the scans.

By default, the operator runs in the `openshift-compliance` namespace, so
make sure all namespaced resources like the deployment or the custom resources
the operator consumes are created there. However, it is possible for the
operator to be deployed in other namespaces as well.

The primary interface towards the Compliance Operator is the
`ComplianceSuite` object, representing a set of scans. The `ComplianceSuite`
can be defined either manually or with the help of `ScanSetting` and
`ScanSettingBinding` objects. Note that while it is possible to use the
lower-level `ComplianceScan` directly as well, it is not recommended.

## Deploying the operator
Before you can actually use the operator, you need to make sure it is
deployed in the cluster. Depending on your needs, you might want to
deploy the upstream released packages or directly from the source.

First, become kubeadmin, either with `oc login` or by exporting `KUBECONFIG`.

### Deploying upstream packages
Deploying from package would deploy the latest released upstream version.

First, create the `CatalogSource` and optionally verify it's been created
successfuly:
```
$ oc create -f deploy/olm-catalog/catalog-source.yaml
$ oc get catalogsource -nopenshift-marketplace
```

Next, create the target namespace and finally either install the operator
from the Web Console or from the CLI following these steps:
```
$ oc create -f deploy/ns.yaml
$ oc create -f deploy/olm-catalog/operator-group.yaml
$ oc create -f deploy/olm-catalog/subscription.yaml
```
The Subscription file can be edited to optionally deploy a custom version,
see the `startingCSV` attribute in the `deploy/olm-catalog/subscription.yaml`
file.

Verify that the expected objects have been created:
```
$ oc get sub -nopenshift-compliance
$ oc get ip -nopenshift-compliance
$ oc get csv -nopenshift-compliance
```

At this point, the operator should be up and running:
```
$ oc get deploy -nopenshift-compliance
$ oc get pods -nopenshift-compliance
```

### Deploying with Helm

The repository contains a [Helm](https://helm.sh/) chart that deploys the
compliance-operator. This chart is currently not published to any official
registries and requires that you [install](https://helm.sh/docs/intro/install/)
Helm version v3.0.0 or greater. You're required to run the chart from this
repository.

Make sure you create the namespace prior to running `helm install`:

```
$ kubectl create -f deploy/ns.yaml
```

Next, deploy a release of the compliance-operator using `helm install` from
`deploy/compliance-operator-chart/`:

```
$ cd deploy/compliance-operator-chart
$ helm install --namespace openshift-compliance --generate-name .
```

The chart defines defaults values in `values.yaml`. You can override these
values in a specific file or supply them to helm using `--set`. For example,
you can run the compliance-operator on EKS using the EKS-specific overrides in
`eks-values.yaml`:

```
$ helm install . --namespace openshift-compliance --generate-name -f eks-values.yaml
```

You can use Helm to uninstall, or delete a release, but Helm does not cleanup
[custom resource
definitions](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#helm).
You must do this manually if you want to remove the custom resource definitions
required by the compliance-operator.

### Deploying from source
```
$ (clone repo)
$ oc create -f deploy/ns.yaml
$ oc project openshift-compliance
$ for f in $(ls -1 deploy/crds/*crd.yaml); do oc apply -f $f -n openshift-compliance; done
$ oc apply -n openshift-compliance -f deploy/
```

### Running the operator locally
If you followed the steps above, the file called `deploy/operator.yaml`
also creates a deployment that runs the operator. If you want to run
the operator from the command line instead, delete the deployment and then
run:

```
make run
```
This is mostly useful for local development.

### Note on namespace removal
Many custom resources deployed with the compliance operators use finalizers
to handle dependencies between objects. If the whole operator namespace gets
deleted (e.g. with `oc delete ns openshift-compliance`), the order of deleting
objects in the namespace is not guaranteed. What can happen is that the
operator itself is removed before the finalizers are processed which would
manifest as the namespace being stuck in the `Terminating` state.

It is recommended to remove all CRs and CRDs prior to removing the namespace
to avoid this issue. The `Makefile` provides a `tear-down` target that does
exactly that.

If the namespace is stuck, you can work around by the issue by hand-editing
or patching any CRs and removing the `finalizers` attributes manually.


## Using the operator

Before starting to use the operator, it's worth checking the descriptions of the
different custom resources it introduces. These definitions are in the
[following document](doc/crds.md)

As part of this guide, it's assumed that you have installed the compliance operator
in the `openshift-compliance` namespace. So you can use:

```
# Set this to the namespace you're deploying the operator at
export NAMESPACE=openshift-compliance
```

There are several profiles that come out-of-the-box as part of the operator
installation.

To view them, use the following command:

```
$ oc get -n $NAMESPACE profiles.compliance
NAME              AGE
ocp4-cis          2m50s
ocp4-cis-node     2m50s
ocp4-e8           2m50s
ocp4-moderate     2m50s
rhcos4-e8         2m46s
rhcos4-moderate   2m46s
```

### Platform and Node scan types
These profiles define different compliance benchmarks and as well as
the scans fall into two basic categories - platform and node. The
platform scans are targetting the cluster itself, in the listing above
they're the `ocp4-*` scans, while the purpose of the node scans is to
scan the actual cluster nodes. All the `rhcos4-*` profiles above can be
used to create node scans.

Before taking one into use, we'll need to configure how the scans
will run. We can do this with the `ScanSetttings` custom resource. The
compliance-operator already ships with a default `ScanSettings` object
that you can take into use immediately:

```
$ oc get -n $NAMESPACE scansettings default -o yaml
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSetting
metadata:
  name: default
  namespace: openshift-compliance
rawResultStorage:
  rotation: 3
  size: 1Gi
roles:
- worker
- master
scanTolerations:
- effect: NoSchedule
  key: node-role.kubernetes.io/master
  operator: Exists
schedule: '0 1 * * *'
```

So, to assert the intent of complying with the `rhcos4-moderate` profile, we can use
the `ScanSettingBinding` custom resource. the example that already exists in this repo
will do just this.

```
$ cat deploy/crds/compliance.openshift.io_v1alpha1_scansettingbinding_cr.yaml
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSettingBinding
metadata:
  name: nist-moderate
profiles:
  - name: ocp4-moderate
    kind: Profile
    apiGroup: compliance.openshift.io/v1alpha1
settingsRef:
  name: default
  kind: ScanSetting
  apiGroup: compliance.openshift.io/v1alpha1
```

To take it into use, do the following:

```
$ oc create -n $NAMESPACE -f deploy/crds/compliance.openshift.io_v1alpha1_scansettingbinding_cr.yaml
scansettingbinding.compliance.openshift.io/nist-moderate created
```

At this point the operator reconciles a `ComplianceSuite` custom resource,
we can use this to track the progress of our scan.

```
$ oc get -n $NAMESPACE compliancesuites -w
NAME            PHASE     RESULT
nist-moderate   RUNNING   NOT-AVAILABLE
```

You can also make use of conditions to wait for a suite to produce results:
```
$ oc wait --for=condition=ready compliancesuite cis-compliancesuite
```

This subsequently creates the `ComplianceScan` objects for the suite.
The `ComplianceScan` then creates scan pods that run on each node in
the cluster. The scan pods execute `openscap-chroot` on every node and
eventually report the results. The scan takes several minutes to complete.

If you're interested in seeing the individual pods, you can do so with:
```
$ oc get -n $NAMESPACE pods -w
```

When the scan is done, the operator changes the state of the ComplianceSuite
object to "Done" and all the pods are transition to the "Completed"
state. You can then check the `ComplianceRemediations` that were found with:
```
$ oc get -n $NAMESPACE complianceremediations
NAME                                                             STATE
workers-scan-auditd-name-format                                  NotApplied
workers-scan-coredump-disable-backtraces                         NotApplied
workers-scan-coredump-disable-storage                            NotApplied
workers-scan-disable-ctrlaltdel-burstaction                      NotApplied
workers-scan-disable-users-coredumps                             NotApplied
workers-scan-grub2-audit-argument                                NotApplied
workers-scan-grub2-audit-backlog-limit-argument                  NotApplied
workers-scan-grub2-page-poison-argument                          NotApplied
```

To apply a remediation, edit that object and set its `Apply` attribute
to `true`:
```
$ oc edit -n $NAMESPACE complianceremediation/workers-scan-no-direct-root-logins
```

The operator then creates a `MachineConfig` object per remediation. This
`MachineConfig` object is rendered to a `MachinePool` and the
`MachineConfigDeamon` running on nodes in that pool pushes the configuration
to the nodes and reboots the nodes.

You can watch the node status with:
```
$ oc get nodes -w
```

Once the nodes reboot, you might want to run another Suite to ensure that
the remediation that you applied previously was no longer found.

## Extracting raw results

The scans provide two kinds of raw results: the full report in the ARF format
and just the list of scan results in the XCCDF format. The ARF reports are,
due to their large size, copied into persistent volumes:
```
$ oc get pv
NAME                                       CAPACITY  CLAIM
pvc-5d49c852-03a6-4bcd-838b-c7225307c4bb   1Gi       openshift-compliance/workers-scan
pvc-ef68c834-bb6e-4644-926a-8b7a4a180999   1Gi       openshift-compliance/masters-scan
$ oc get pvc
NAME                     STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
ocp4-moderate            Bound    pvc-01b7bd30-0d19-4fbc-8989-bad61d9384d9   1Gi        RWO            gp2            37m
rhcos4-with-usb-master   Bound    pvc-f3f35712-6c3f-42f0-a89a-af9e6f54a0d4   1Gi        RWO            gp2            37m
rhcos4-with-usb-worker   Bound    pvc-7837e9ba-db13-40c4-8eee-a2d1beb0ada7   1Gi        RWO            gp2            37m
```

An example of extracting ARF results from a scan called `workers-scan` follows:

Once the scan had finished, you'll note that there is a `PersistentVolumeClaim` named
after the scan:
```
oc get pvc/workers-scan
NAME            STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
workers-scan    Bound    pvc-01b7bd30-0d19-4fbc-8989-bad61d9384d9   1Gi        RWO            gp2            38m
```
You'll want to start a pod that mounts the PV, for example:
```yaml
apiVersion: "v1"
kind: Pod
metadata:
  name: pv-extract
spec:
  containers:
    - name: pv-extract-pod
      image: registry.access.redhat.com/ubi8/ubi
      command: ["sleep", "3000"]
      volumeMounts:
        - mountPath: "/workers-scan-results"
          name: workers-scan-vol
  volumes:
    - name: workers-scan-vol
      persistentVolumeClaim:
        claimName: workers-scan
```

You can inspect the files by listing the `/workers-scan-results` directory and copy the
files locally:
```
$ oc exec pods/pv-extract -- ls /workers-scan-results/0
lost+found
workers-scan-ip-10-0-129-252.ec2.internal-pod.xml.bzip2
workers-scan-ip-10-0-149-70.ec2.internal-pod.xml.bzip2
workers-scan-ip-10-0-172-30.ec2.internal-pod.xml.bzip2
$ oc cp pv-extract:/workers-scan-results .
```
The files are bzipped. To get the raw ARF file:
```
$ bunzip2 -c workers-scan-ip-10-0-129-252.ec2.internal-pod.xml.bzip2 > workers-scan-ip-10-0-129-252.ec2.internal-pod.xml
```

The XCCDF results are much smaller and can be stored in a configmap, from
which you can extract the results. For easier filtering, the configmaps
are labeled with the scan name:
```
$ oc get cm -l=compliance.openshift.io/scan-name=masters-scan
NAME                                            DATA   AGE
masters-scan-ip-10-0-129-248.ec2.internal-pod   1      25m
masters-scan-ip-10-0-144-54.ec2.internal-pod    1      24m
masters-scan-ip-10-0-174-253.ec2.internal-pod   1      25m
```

To extract the results, use:
```
$ oc extract cm/masters-scan-ip-10-0-174-253.ec2.internal-pod
```

Note that if the results are too big for the ConfigMap, they'll be bzipped and
base64 encoded.

## OS support

### Node scans

Note that the current testing has been done in RHCOS. In the absence of
RHEL/CentOS support, one can simply run OpenSCAP directly on the nodes.

### Platform scans

Current testing has been done on OpenShift (OCP). The project is open to
getting other platforms tested, so volunteers are needed for this.

The current supported versions of OpenShift are 4.6 and up.

## Additional documentation

See the [self-paced workshop](doc/tutorials/README.md) for a hands-on tutorial,
including advanced topics such as content building.

## Must-gather support

An `oc adm must-gather` image for collecting operator information for debugging
or support is available at `quay.io/pravin_dsilva/must-gather:latest`:

```
$ oc adm must-gather --image=quay.io/pravin_dsilva/must-gather:latest
```

## Metrics
The compliance-operator exposes the following metrics to Prometheus when cluster-monitoring is available.

    # HELP compliance_operator_compliance_remediation_status_total A counter for the total number of updates to the status of a ComplianceRemediation
    # TYPE compliance_operator_compliance_remediation_status_total counter
    compliance_operator_compliance_remediation_status_total{name="remediation-name",state="NotApplied"} 1

    # HELP compliance_operator_compliance_scan_status_total A counter for the total number of updates to the status of a ComplianceScan
    # TYPE compliance_operator_compliance_scan_status_total counter
    compliance_operator_compliance_scan_status_total{name="scan-name",phase="AGGREGATING",result="NOT-AVAILABLE"} 1

    # HELP compliance_operator_compliance_scan_error_total A counter for the total number of encounters of error
    # TYPE compliance_operator_compliance_scan_error_total counter
    compliance_operator_compliance_scan_error_total{name="scan-name",error="some_error"} 1

    # HELP compliance_operator_compliance_state A gauge for the compliance state of a ComplianceSuite. Set to 0 when COMPLIANT, 1 when NON-COMPLIANT, 2 when INCONSISTENT, and 3 when ERROR
    # TYPE compliance_operator_compliance_state gauge
    compliance_operator_compliance_state{name="some-compliance-suite"} 1

After logging into the console, navigating to Monitoring -> Metrics, the compliance_operator* metrics can be queried using the metrics dashboard. The `{__name__=~"compliance.*"}` query can be used to view the full set of metrics.

Testing for the metrics from the cli can also be done directly with a pod that curls the metrics service. This is useful for troubleshooting.

```
oc run --rm -i --restart=Never --image=registry.fedoraproject.org/fedora-minimal:latest -n openshift-compliance metrics-test -- bash -c 'curl -ks -H "Authorization: Bea
rer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" https://metrics.openshift-compliance.svc:8585/metrics-co' | grep compliance
```

## Contributor Guide

### Writing Release Notes

Release notes are maintained in the [changelog](CHANGELOG.md) and follow
guidelines based on [keep a changelog](https://keepachangelog.com/en/1.0.0/).
This section describes additional guidelines, conventions, and the overall
process for writing release notes.

#### Guidelines

* Each release should contain release notes
* Changes should be applicable to at least one of the six types listed below
* Use literals for code and configuration (e.g. `defaultScanSettingsSchedule`
  or `nodeSelector`)
* Write your notes with users as the audience
* Link to additional documentation
  - Bug fixes should link to bug reports (GitHub Issues or Jira items)
  - Features or enhancements should link to RFEs (GitHub Issues or Jira items)
* Use active voice
  - Active voice is more direct and concise than passive voice, perfect for
    release notes
  - Focus on telling the user how a change will affect them
  - Examples
    - *You can now adjust the frequency of your scans by...*
    - *The compliance-operator no longer supports...*

#### Change Types

The following describe each potential section for a release changelog.

1. Enhancements
2. Fixes
3. Internal Changes
4. Deprecations
5. Removals
6. Security

*Enhancements* are reserved for communicating any new features or
functionality. You should include any new configuration or processes a user
needs to take to use the new feature.

*Fixes* are for noting improvements to any existing functionality.

*Internal Changes* are ideal for communicating refactors not exposed to end
users. Even if a change does not directly impact end users, it is still
important to highlight paying down technical debt and the rationale for those
changes, especially since they impact the project's roadmap.

*Deprecations* is for any functionality, feature, or configuration that is
being deprecated and staged for removal. Deprecations should include why we're
preparing to remove the functionality and signal any suitable replacements
users should adopt.

*Removals* is for any functionality, feature, or configuration that is being
removed. Typically, entries in this section will have been deprecated for some
period of time. The compliance-operator follows the
[Kubernetes deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

*Security* is reserved for communicating security fixes and remediations for
CVEs.

A change can apply to multiple change types. For example, a bug fix for a CVE
should be mentioned in the *Fixes* and *Security* sections.

#### Process

Contributors must include a release note with their changes. New notes should
be added to the [Unreleased section](CHANGELOG.md#unreleased) of the
[changelog](CHANGELOG.md). Reviewers will assess the accuracy of the release
note against the change.

Maintainers preparing a new release will propose a change that renames the
[Unreleased release notes](CHANGELOG.md#unreleased) to the newly released
version and release date. Maintainers can remove empty sections if it does not
contain any release notes for a specific release.

Maintainers will remove the content of the [Unreleased section](CHANGELOG.md#unreleased)
to allow for new release notes for the next release.

Following this process makes it easier to maintain and release accurate release
notes without having to retroactively write release notes for merged changes.

#### Examples

The following is an example release note for a feature with a security note.

```
## Unreleased
### Enhancements

- Allow configuring result servers using `nodeSelector` and `tolerations`
  ([RFE](https://github.com/openshift/compliance-operator/issues/696))
  - You can now specify which nodes to use for storing raw compliance results
    using the `nodeSelector` and `tolerations` from `ScanSettings`.
  - By default, raw results are stored on nodes labeled
    `node-role.kubernetes.io/master`.
  - Please refer to the upstream Kubernetes
    [documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/)
    for details on how to use `nodeSelector` and `tolerations`.

### Security

- Allow configuring result servers using `nodeSelector` and `tolerations`
  ([RFE](https://github.com/openshift/compliance-operator/issues/696))
  - Raw compliance results may contain sensitive information about the
    deployment, its infrastructure, or applications. Make sure you send raw
    results to trusted nodes.
```

### Proposing Releases

The release process is separated into three phases, with dedicated `make`
targets. All targets require that you supply the `OPERATOR_VERSION` prior to
running `make`, which should be a semantic version formatted string (e.g.,
`OPERATOR_VERSION=0.1.49`).

#### Preparing the Release

The first phase of the release process is preparing the release locally. You
can do this by running the `make prepare-release` target. All changes are
staged locally. This is intentional so that you have the opportunity to
review the changes before proposing the release in the next step.

#### Proposing the Release

The second phase of the release is to push the release to a dedicated branch
against the origin repository. You can perform this step using the `make
push-release` target.

Please note, this step makes changes to the upstream repository, so it is
imperative that you review the changes you're committing prior to this step.
This steps also requires that you have necessary permissions on the repository.

#### Releasing Images

The third and final step of the release is to build new images and push them to
an offical image registry. You can build new images and push using `make
release-images`. Note that this operation also requires you have proper
permissions on the remote registry.
