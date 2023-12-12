FROM centos:7 as build

ARG version
ARG commit

RUN yum install -y rpm-build make

ENV GOLANG_VERSION 1.20
RUN ARCH=$(uname -m) && ARCH_GO=$(echo $ARCH | sed 's/x86_64/amd64/;s/arm.*/arm64/;s/aarch64/arm64/') && \
  curl -sSL https://dl.google.com/go/go${GOLANG_VERSION}.linux-${ARCH_GO}.tar.gz \
  | tar -C /usr/local -xz
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN mkdir -p /root/rpmbuild/{SPECS,SOURCES}

COPY gpu-admission.spec /root/rpmbuild/SPECS
COPY gpu-admission-source.tar.gz /root/rpmbuild/SOURCES

RUN echo '%_topdir /root/rpmbuild' > /root/.rpmmacros \
  && echo '%__os_install_post %{nil}' >> /root/.rpmmacros \
  && echo '%debug_package %{nil}' >> /root/.rpmmacros
WORKDIR /root/rpmbuild/SPECS
RUN rpmbuild -ba  \
  --define 'version '${version}'' \
  --define 'commit '${commit}'' \
  gpu-admission.spec


FROM centos:7

ARG version
ARG commit

COPY --from=build /root/rpmbuild/RPMS/*/gpu-admission-${version}-${commit}.el7.*.rpm /tmp

RUN rpm -ivh /tmp/gpu-admission-${version}-${commit}.el7.*.rpm

EXPOSE 3456

CMD ["/bin/bash", "-c", "/usr/bin/gpu-admission --address=0.0.0.0:3456  --logtostderr=true $EXTRA_FLAGS"]
