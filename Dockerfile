FROM busybox:latest

MAINTAINER dewang wei <dewang.wei@gmail.com>


WORKDIR /opt

ADD builds/linux_amd64/go-transit /opt/bin/
ADD etc/*.json /opt/etc/

EXPOSE 9000

VOLUME ["/opt/etc","/data"]

CMD ["/opt/bin/go-transit","-f","/opt/etc/config.json"]