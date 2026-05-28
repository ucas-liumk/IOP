package com.gallant.collab.config;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.gallant.collab.common.ApiResponse;
import org.springframework.core.MethodParameter;
import org.springframework.http.MediaType;
import org.springframework.http.converter.HttpMessageConverter;
import org.springframework.http.server.ServerHttpRequest;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.lang.NonNull;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.servlet.mvc.method.annotation.ResponseBodyAdvice;

/** 自动把 Controller 返回值包成 ApiResponse, 避免每个方法都写 ApiResponse.ok() */
@RestControllerAdvice(basePackages = "com.gallant.collab.controller")
public class ResponseWrapAdvice implements ResponseBodyAdvice<Object> {

    private final ObjectMapper objectMapper;

    public ResponseWrapAdvice(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    @Override
    public boolean supports(@NonNull MethodParameter returnType,
                            @NonNull Class<? extends HttpMessageConverter<?>> converterType) {
        return !ApiResponse.class.isAssignableFrom(returnType.getParameterType());
    }

    @Override
    public Object beforeBodyWrite(Object body, @NonNull MethodParameter returnType,
                                  @NonNull MediaType selectedContentType,
                                  @NonNull Class<? extends HttpMessageConverter<?>> selectedConverterType,
                                  @NonNull ServerHttpRequest request,
                                  @NonNull ServerHttpResponse response) {
        // 二进制下载等不包
        if (body instanceof byte[] || body instanceof org.springframework.core.io.Resource) return body;
        // String 类型的响应需要手动序列化, 否则 Spring 会按 String 处理直接写到响应体
        if (body instanceof String) {
            try {
                return objectMapper.writeValueAsString(ApiResponse.ok(body));
            } catch (Exception e) {
                return body;
            }
        }
        return ApiResponse.ok(body);
    }
}
