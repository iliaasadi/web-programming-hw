# تمرین عملی HTTP

### ۱. اجرای سرویس فقط روی شبکه داخلی

سرویس قدیمی <span dir="ltr">authentication</span> ندارد. برای استفاده داخل سازمان این موضوع مشکلی نیست، ولی اگر پورتش از اینترنت باز باشد بقیه هم می‌توانند از آن استفاده کنند. پس سرویس می‌تواند روشن باشد، به شرطی که از بیرون به آن پورت وصل نشوند.

اگر فقط همان ماشین به سرویس نیاز دارد، آن را به <span dir="ltr">127.0.0.1</span> وصل می‌کنم نه به <span dir="ltr">0.0.0.0</span>. حالت دوم روی همه کارت‌های شبکه گوش می‌دهد، حتی کارتی که بیرون از شبکه داخلی است. وقتی چند ماشین داخل سازمان باید سرویس را ببینند، به‌جای این از یک IP خصوصی استفاده می‌کنم.

در <span dir="ltr">Docker</span> هم پورت سرویس را <span dir="ltr">publish</span> نمی‌کنم. <span dir="ltr">httpbin</span> را روی یک شبکه داخلی می‌گذارم و فقط پروکسی عضو همان شبکه است. از بیرون فقط پورت ۸۰ خود پروکسی باز می‌ماند و درخواست‌ها از همان‌جا به سرویس داخلی می‌روند. عدد ۸۰۸۱ در صورت تمرین مثال است. ایمیج <span dir="ltr">go-httpbin</span> داخل کانتینر معمولاً روی ۸۰۸۰ گوش می‌دهد و این پورت را هم به <span dir="ltr">host</span> وصل نمی‌کنم.

### ۲. TLS Termination

این قسمت را برای تحویل عملی پیاده نمی‌کنم. چیزی که بالا می‌آید <span dir="ltr">HTTP</span> روی پورت ۸۰ است.

<span dir="ltr">TLS Termination</span> یعنی رمزنگاری سر مرز شبکه، روی <span dir="ltr">reverse proxy</span>، تمام شود. کاربر بیرون با پروکسی <span dir="ltr">HTTPS</span> صحبت می‌کند و گواهی دست پروکسی است. پروکسی درخواست را باز می‌کند و چون مسیر تا سرویس قدیمی داخل شبکه خودمان است، ادامه را با <span dir="ltr">HTTP</span> معمولی می‌فرستد. برنامه <span dir="ltr">legacy</span> معمولاً گواهی را درست نگه نمی‌دارد و برای این کار لازم نیست خودش تغییر کند.

این روش جای <span dir="ltr">JWT</span> را نمی‌گیرد. <span dir="ltr">TLS</span> جلوی خوانده شدن درخواست وسط راه را می‌گیرد، ولی هنوز باید معلوم شود چه کسی اجازه دسترسی دارد. اگر <span dir="ltr">TLS</span> تا خود سرویس قدیمی برود، پروکسی هدر و بدنه را نمی‌بیند و نمی‌تواند <span dir="ltr">JWT</span> را بررسی کند.

### ۳. پروکسی

کد پروکسی در <span dir="ltr">proxy/main.go</span> است. روی پورت ۸۰ گوش می‌دهد و فقط <span dir="ltr">HS256</span> را قبول می‌کند. هدر باید این باشد: <span dir="ltr">Authorization: Bearer &lt;token&gt;</span>. اگر توکن نباشد، امضا با <span dir="ltr">JWT_SECRET</span> نخواند، یا <span dir="ltr">exp</span> گذشته باشد، جواب <span dir="ltr">401</span> می‌دهم و درخواست به <span dir="ltr">httpbin</span> نمی‌رود. <span dir="ltr">method</span> و مسیر و <span dir="ltr">body</span> درخواست معتبر همان‌طور به <span dir="ltr">httpbin</span> فرستاده می‌شود.

در <span dir="ltr">docker-compose</span> دو سرویس هست. <span dir="ltr">httpbin</span> ایمیج <span dir="ltr">ghcr.io/mccutchen/go-httpbin</span> است و فقط به شبکه <span dir="ltr">internal</span> وصل است. این شبکه به بیرون راه ندارد و پورت ۸۰۸۰ روی <span dir="ltr">host</span> باز نشده. پروکسی به شبکه <span dir="ltr">edge</span> هم وصل است تا از سیستم خودم فقط پورت ۸۰ دیده شود، و درخواست درست را به <span dir="ltr">http://httpbin:8080</span> بفرستد.

### ۴. اجرا

کنار همین پوشه فایل <span dir="ltr">.env</span> را از روی نمونه می‌سازم و داخلش <span dir="ltr">JWT_SECRET</span> را می‌گذارم. خود <span dir="ltr">.env</span> وارد گیت نمی‌شود.

```bash
cd http-hands-on
cp .env.example .env
docker compose up --build -d
```

بدون توکن جواب <span dir="ltr">401</span> است:

```bash
curl -i http://127.0.0.1/get
```

برای توکن معتبر، <span dir="ltr">exp</span> باید جلوتر از الان باشد. پروکسی خودش توکن نمی‌سازد.

```bash
set -a && source .env && set +a
TOKEN=$(python3 - << 'END'
import os, time, json, hmac, hashlib, base64
def b64(raw):
    return base64.urlsafe_b64encode(raw).rstrip(b'=')
header = b64(json.dumps({'alg': 'HS256', 'typ': 'JWT'}, separators=(',', ':')).encode())
payload = b64(json.dumps({'exp': int(time.time()) + 3600}, separators=(',', ':')).encode())
sig = b64(hmac.new(os.environ['JWT_SECRET'].encode(), header + b'.' + payload, hashlib.sha256).digest())
print((header + b'.' + payload + b'.' + sig).decode())
END
)
curl -i http://127.0.0.1/get -H "Authorization: Bearer $TOKEN"
```

این را امتحان کردم. بدون توکن <span dir="ltr">401</span> آمد و با توکن درست <span dir="ltr">200</span>.

### ۵. فایل‌ها

```text
http-hands-on/
  docker-compose.yml
  .env.example
  .gitignore
  proxy/Dockerfile
  proxy/go.mod
  proxy/main.go
```

برای فهم صورت تمرین، مرتب کردن همین متن، و بخشی از کد پروکسی از هوش مصنوعی کمک گرفتم.
