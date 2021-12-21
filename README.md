<h1>Build on Linux:</h1>

git clone https://github.com/MaxZamaliev/nginx-log-exporter.git<br>
cd nginx-log-exporter<br>
export GOPATH=\`pwd\`<br>
go get github.com/hpcloud/tail<br>
go get github.com/prometheus/client_golang/prometheus<br>
go get github.com/prometheus/client_golang/prometheus/promhttp<br>
go build nginx-log-exporter.go<br>


<h1>Prepare nginx:</h1>
Add to your nginx.conf file:<br>
&nbsp;&nbsp;geoip_country /usr/share/GeoIP/GeoIP.dat;<br>
&nbsp;&nbsp;log_format custom  '$remote_addr ($geoip_country_code) - $remote_user [$time_local] "$host" "$request" '<br>
&nbsp;&nbsp;&nbsp;&nbsp;'$status $body_bytes_sent $request_time "$http_referer" '<br>
&nbsp;&nbsp;&nbsp;&nbsp;'"$http_user_agent" "$http_x_forwarded_for"';<br>
&nbsp;&nbsp;access_log /var/log/nginx/access.log custom;<br>


<h1>Install on CentOS 8:</h1>

<b>1. Copy binary file nginx-log-exporter to /usr/local/bin/</b>

<b>2. Create user:</b>

useradd -M -s /bin/false nginx_exporter

<b>3. Create file `/etc/systemd/system/nginx-log_exporter.service` with text:</b>

[Unit]<br>
Description=Prometheus nginx log Exporter<br>
Wants=network-online.target<br>
After=network-online.target<br>
<br>
[Service]<br>
User=nginx_exporter<br>
Group=nginx_exporter<br>
Type=simple<br>
ExecStart=/usr/local/bin/nginx-log-exporter<br>
<br>
[Install]<br>
WantedBy=multi-user.target<br>

<b>4. Enable and start service:</b>

systemctl enable --now nginx-log-exporter

<b>5. Test service:</b>

curl http://localhost:9113/metrics
