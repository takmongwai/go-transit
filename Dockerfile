FROM busybox:latest

MAINTAINER dewang wei <dewang.wei@gmail.com>


WORKDIR /opt

COPY builds/linux_amd64/go-transit /opt/bin/
COPY etc/config.json /opt/etc/
COPY etc/access_users.json /opt/etc/


EXPOSE 9000

VOLUME ["/opt/etc","/data"]

CMD ["/opt/bin/go-transit","-f","/opt/etc/config.json","-u","/opt/etc/access_users.json"]