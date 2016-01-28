
kernel void add(
                global int* a,
                global int* b,
                global int* res_mem) {

    size_t n = get_global_size(0);
    size_t x = get_global_id(0);
    size_t y = get_global_id(1);

    int sum = 0;
    for (int i = 0; i < n; ++i) {
        sum += a[y * n + i] * b[i * n + x];
    }
    
    res_mem[y * n + x] = sum;
//    res_mem[y * n + x] = y * n + x; // + b[y * n + x];
}
