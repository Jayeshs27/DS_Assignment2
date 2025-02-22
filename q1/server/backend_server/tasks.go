package main

func isPrime(n int64) bool {
    if n < 2 {
        return false
    }
    for i := int64(2); i*i <= n; i++ { 
        if n%i == 0 {
            return false
        }
    }
    return true
}

func nthPrime(n int64) int64 {
    if n < 1 {
        return -1 
    }
	cnt, num := int64(0), int64(1)
    for cnt < n {
        num++
        if isPrime(num) {
            cnt++
        }
    }
    return num
}

func fibonacci(n int64) int64{
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}

func Sum(n int64) int64 {
	sum := int64(0)
	for i:= int64(0) ; i < n ; i++{
		sum += (i + 1)
	}
	return sum
}

func executeTask(tasktype int32, n int64) (int64){
	if tasktype == 0 {
		return Sum(n)
	} else if tasktype == 1 {
		return nthPrime(n)
	} else {
		return fibonacci(n)
	}
}