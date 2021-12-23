# 基于hostPort的LoadBalancer控制器

# 说明
实现该负载均衡控制器的目的是为了支持如下场景
- 裸金属服务器组成的k8s集群.
- 每台边缘节点都有自己的公网IP
- 您需要一个域名使用通配符解析，将某个通配符域名解析到每一个边缘节点

## 安装
> 本次将使用 .lb.kunstack.com 作为通配符域名进行测试, 清单文件中也是这个域名，请在实际使用前做替换

```shell
# 下载yaml清单文件
wget https://raw.githubusercontent.com/kunstack/pharos/main/deploy/lb.manifest.yaml
# 在您使用时需要替换清单文件中的域名后缀 .lb.kunstack.com 为你的通配符域名
sed -i 's/\.lb\.kunstack\.com/\.yourdomain\.com/g' lb.manifest.yaml

kubectl apply -f  ./lb.manifest.yaml

# 安装后确保所有的pod均处于Running状态
kubectl -n  kube-system get pods -l app=pharos-lb-controller
```

## 验证
如果一切工作良好，您将看到所有类型为 LoadBalancer 的service都将分配一个 uuid+通配符域名后缀的 EXTERNAL-IP

```shell
[root@10-23-51-13 ~]# kubectl -n kubesphere-controls-system get svc kubesphere-router-kubesphere-system
NAME                                  TYPE           CLUSTER-IP       EXTERNAL-IP                                            PORT(S)                      AGE
kubesphere-router-kubesphere-system   LoadBalancer   172.16.194.182   1147990d-378a-4cc6-9d30-cb35a9b6339a.lb.kunstack.com   80:32328/TCP,443:31850/TCP   5d5h
```
