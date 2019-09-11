# vim: tabstop=4 shiftwidth=4 expandtab
%global _hardened_build 1
# Debug package is empty anyway
%define debug_package %{nil}

%global _release        1
%global provider        github
%global provider_tld    com
%global project         m13253
%global repo            dns-over-https
%global provider_prefix %{provider}.%{provider_tld}/%{project}/%{repo}
%global import_path     %{provider_prefix}

#define commit          984df34ca7b45897ecb5871791e398cc160a4b93

%if 0%{?commit:1}
%define shortcommit     %(c=%{commit}; echo ${c:0:7})
%define _date           %(date +'%%Y%%m%%dT%%H%%M%%S')
%endif

%define rand_id         %(head -c20 /dev/urandom|od -An -tx1|tr -d '[[:space:]]')

%if ! 0%{?gobuild:1}
%define gobuild(o:) go build -ldflags "${LDFLAGS:-} -B 0x%{rand_id}" -a -v -x %{?**};
%endif

%if ! 0%{?gotest:1}
%define gotest() go test -ldflags "${LDFLAGS:-}" %{?**}
%endif

Name:           %{repo}
Version:        2.1.2
%if 0%{?commit:1}
Release:    %{_release}.git%{shortcommit}.%{_date}%{?dist}
Source0:        https://%{import_path}/archive/%{commit}.tar.gz
%else
Release:        %{_release}%{?dist}
Source0:        https://%{import_path}/archive/v%{version}.tar.gz
%endif
Patch0:         %{name}-%{version}-systemd.patch

Summary:        High performance DNS over HTTPS client & server
License:        MIT
URL:            https://github.com/m13253/dns-over-https


# e.g. el6 has ppc64 arch without gcc-go, so EA tag is required
# If go_compiler is not set to 1, there is no virtual provide. Use golang instead.
#BuildRequires:  %{?go_compiler:compiler(go-compiler)}%{!?go_compiler:golang} >= 1.10
BuildRequires:  golang >= 1.10
BuildRequires:  systemd
BuildRequires:  upx

%description
%{summary}

%package common
BuildArch: noarch
Summary: %{summary} - common files

%description common
%{summary}

%package server
ExclusiveArch:  %{?go_arches:%{go_arches}}%{!?go_arches:%{ix86} x86_64 %{arm}}
Summary: %{summary} - Server
Requires(pre):          shadow-utils
Requires(post):         systemd
Requires(preun):        systemd
Requires(postun):       systemd

%description server
%{summary}

%package client
ExclusiveArch:  %{?go_arches:%{go_arches}}%{!?go_arches:%{ix86} x86_64 %{arm}}
Summary: %{summary} - Client
Requires(pre):          shadow-utils
Requires(post):         systemd
Requires(preun):        systemd
Requires(postun):       systemd

%description client
%{summary}

%package selinux
BuildArch:      noarch

Source3:        doh_server.fc
Source4:        doh_server.if
Source5:        doh_server.te
Source6:        doh_client.fc
Source7:        doh_client.if
Source8:        doh_client.te

BuildRequires:  selinux-policy
BuildRequires:  selinux-policy-devel
Requires:       %{name}

Requires(post):     policycoreutils
Requires(post):     policycoreutils-python 
Requires(postun):   policycoreutils

Summary: SELinux policy for %{name}

%description selinux
%summary
 
%prep
%if 0%{?commit:1}
%autosetup -n %{name}-%{commit} -p1
%else
%autosetup -n %{name}-%{version} -p1
%endif

mkdir -p selinux
cp %{SOURCE3} %{SOURCE4} %{SOURCE5} %{SOURCE6} %{SOURCE7} %{SOURCE8} selinux

%build
cd selinux
make -f /usr/share/selinux/devel/Makefile doh_server.pp doh_client.pp || exit
cd -

%set_build_flags
%make_build \
    PREFIX=%{_prefix} \
    GOBUILD="go build -ldflags \"-s -w -B 0x%{rand_id}\" -a -v -x"

%install
%make_install \
    PREFIX=%{_prefix}
install -Dpm 0600 selinux/doh_server.pp %{buildroot}%{_datadir}/selinux/packages/doh_server.pp
install -Dpm 0644 selinux/doh_server.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/doh_server.if
install -Dpm 0600 selinux/doh_client.pp %{buildroot}%{_datadir}/selinux/packages/doh_client.pp
install -Dpm 0644 selinux/doh_client.if %{buildroot}%{_datadir}/selinux/devel/include/contrib/doh_client.if

mkdir -p %{buildroot}%{_docdir}/%{name}
mv %{buildroot}%{_sysconfdir}/%{name}/*.example %{buildroot}%{_docdir}/%{name}

mkdir -p %{buildroot}%{_libdir}
mv %{buildroot}%{_sysconfdir}/NetworkManager %{buildroot}%{_libdir}/

for i in $(find %{_buildroot}%{_bindir} -type f)
do
    upx $i
done

%files common
%license LICENSE
%doc Changelog.md Readme.md

%files server
%{_libdir}/NetworkManager/dispatcher.d/doh-server
%{_docdir}/%{name}/doh-server.conf.example
%config(noreplace) %{_sysconfdir}/%{name}/doh-server.conf
%{_bindir}/doh-server
%{_unitdir}/doh-server.service

%files client
%{_libdir}/NetworkManager/dispatcher.d/doh-client
%{_docdir}/%{name}/doh-client.conf.example
%config(noreplace) %{_sysconfdir}/%{name}/doh-client.conf
%{_bindir}/doh-client
%{_unitdir}/doh-client.service

%pre server
test -d %{_sharedstatedir}/home || mkdir -p %{_sharedstatedir}/home
getent group doh-server > /dev/null || groupadd -r doh-server
getent passwd doh-server > /dev/null || \
    useradd -r -d %{_sharedstatedir}/home/doh-server -g doh-server \
    -s /sbin/nologin -c "%{name} - server" doh-server
exit 0

%pre client
test -d %{_sharedstatedir}/home || mkdir -p %{_sharedstatedir}/home
getent group doh-client > /dev/null || groupadd -r doh-client
getent passwd doh-client > /dev/null || \
    useradd -r -d %{_sharedstatedir}/home/doh-client -g doh-client \
    -s /sbin/nologin -c "%{name} - client" doh-client
exit 0

%post server
%systemd_post doh-server.service

%preun server
%systemd_preun doh-server.service

%postun server
%systemd_postun_with_restart doh-server.service

%post client
%systemd_post doh-client.service

%preun client
%systemd_preun doh-client.service

%postun client
%systemd_postun_with_restart doh-client.service

%files selinux
%{_datadir}/selinux/packages/doh_server.pp
%{_datadir}/selinux/devel/include/contrib/doh_server.if
%{_datadir}/selinux/packages/doh_client.pp
%{_datadir}/selinux/devel/include/contrib/doh_client.if

%post selinux
semodule -n -i %{_datadir}/selinux/packages/doh_server.pp
semodule -n -i %{_datadir}/selinux/packages/doh_client.pp
if /usr/sbin/selinuxenabled ; then
    /usr/sbin/load_policy
    /usr/sbin/fixfiles -R %{name}-server restore
    /usr/sbin/fixfiles -R %{name}-client restore
fi;
semanage -i - << __eof
port -a -t doh_server_port_t -p tcp "8053"
port -a -t doh_client_port_t -p udp "5380"
__eof
exit 0

%postun selinux
if [ $1 -eq 0 ]; then
    semanage -i - << __eof
port -d -t doh_server_port_t -p tcp "8053"
port -d -t doh_client_port_t -p udp "5380"
__eof

    semodule -n -r doh_server
    semodule -n -r doh_client
    if /usr/sbin/selinuxenabled ; then
       /usr/sbin/load_policy
       /usr/sbin/fixfiles -R %{name}-server restore
       /usr/sbin/fixfiles -R %{name}-client restore
    fi;
fi;
exit 0

%changelog
* Tue Sep 10 2019 fuero <fuerob@gmail.com> 2.1.2-1
- initial package

