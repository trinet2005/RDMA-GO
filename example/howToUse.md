首先使用源码中的`wrapper.c` `wrapper.h`编译出动态编译库

```
gcc -c -o wrapper.o wrapper.c
gcc -shared -o libwrapper.so wrapper.o
```

获得libwrapper.so后 放入编译的根目录 并在调用库的代码处添加以下import

```go
/*
#cgo LDFLAGS: -libverbs -L. -lwrapper
*/
import "C"
```

注意 `import "C"`必须和正常的import直接没有多余空行