# Troubleshooting

##  Move2kube Installation

### Permissions of the `go` folder

```
user@host:~/go/src/move2kube$ make build
cp: cannot create regular file '/home/user/go/bin/': Not a directory
Makefile:70: recipe for target '/home/user/go/src/move2kube/bin/move2kube' failed
make: *** [/home/user/go/src/move2kube/bin/move2kube] Error 1
```

Solution: Make sure the `go` folder is writable

### Test fail


```
user@host:~/go/src/move2kube$ make test
...
FAIL
FAIL        github.com/konveyor/move2kube/internal/source        1.580s
?           github.com/konveyor/move2kube/internal/source/compose        [no test files]
?           github.com/konveyor/move2kube/internal/source/data        [no test files]
?           github.com/konveyor/move2kube/internal/transformer        [no test files]
?           github.com/konveyor/move2kube/internal/transformer/templates        [no test files]
ok          github.com/konveyor/move2kube/internal/types        0.214s
?           github.com/konveyor/move2kube/samples/golang        [no test files]
?           github.com/konveyor/move2kube/types        [no test files]
ok          github.com/konveyor/move2kube/types/collection        0.106s
ok          github.com/konveyor/move2kube/types/info        0.095s
ok          github.com/konveyor/move2kube/types/output        0.277s
ok          github.com/konveyor/move2kube/types/plan        0.362s
ok          github.com/konveyor/move2kube/types/qaengine        0.068s
FAIL

Makefile:89: recipe for target 'test' failed
```
Inspecting the debug output found:

```
time="2020-09-23T02:53:58Z" level=debug msg="Docker not supported : exit status 126 : docker: Got permission denied while trying to connect to the Docker daemon socket at unix:///var/run/docker.sock: Post http://%2Fvar%2Frun%2Fdocker.sock/v1.40/containers/create: dial unix /var/run/docker.sock: connect: permission denied.\nSee 'docker run --help'.\n"
```

Solution: Make sure the current user is part of the docker group:

```
sudo usermod -aG docker <username>
```

