package com.gallant.collab.config;

import com.gallant.collab.common.UserContext;
import com.gallant.collab.domain.AppUser;
import com.gallant.collab.mapper.AppUserMapper;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;
import org.springframework.lang.NonNull;
import org.springframework.web.servlet.HandlerInterceptor;
import org.springframework.web.servlet.config.annotation.CorsRegistry;
import org.springframework.web.servlet.config.annotation.InterceptorRegistry;
import org.springframework.web.servlet.config.annotation.WebMvcConfigurer;

@Configuration
public class WebConfig implements WebMvcConfigurer {

    private final AppUserMapper userMapper;
    @Value("${collab.mock-user-id:1}")
    private Long mockUserId;

    public WebConfig(AppUserMapper userMapper) {
        this.userMapper = userMapper;
    }

    @Override
    public void addCorsMappings(@NonNull CorsRegistry registry) {
        registry.addMapping("/**")
                .allowedOriginPatterns("*")
                .allowedMethods("*")
                .allowedHeaders("*")
                .exposedHeaders("X-Total-Count")
                .allowCredentials(true)
                .maxAge(3600);
    }

    @Override
    public void addInterceptors(@NonNull InterceptorRegistry registry) {
        registry.addInterceptor(new HandlerInterceptor() {
            @Override
            public boolean preHandle(@NonNull HttpServletRequest req,
                                     @NonNull HttpServletResponse resp,
                                     @NonNull Object handler) {
                String h = req.getHeader("X-User-Id");
                Long uid = mockUserId;
                if (h != null && !h.isBlank()) {
                    try { uid = Long.parseLong(h); } catch (NumberFormatException ignore) {}
                }
                AppUser u = userMapper.selectById(uid);
                if (u == null) u = userMapper.selectById(mockUserId);
                if (u == null) {
                    // 兜底兼容首次启动尚未灌入种子数据的情况
                    UserContext.set(new UserContext.CurrentUser(0L, "系统", "系统"));
                } else {
                    UserContext.set(new UserContext.CurrentUser(u.getId(), u.getName(), u.getDept()));
                }
                return true;
            }

            @Override
            public void afterCompletion(@NonNull HttpServletRequest req,
                                        @NonNull HttpServletResponse resp,
                                        @NonNull Object handler, Exception ex) {
                UserContext.clear();
            }
        }).addPathPatterns("/**");
    }
}
