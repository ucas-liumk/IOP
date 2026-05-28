package com.gallant.collab.controller;

import com.gallant.collab.common.UserContext;
import com.gallant.collab.domain.AppUser;
import com.gallant.collab.mapper.AppUserMapper;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/users")
@RequiredArgsConstructor
public class UserController {

    private final AppUserMapper mapper;

    @GetMapping
    public List<AppUser> list() {
        return mapper.selectList(null);
    }

    @GetMapping("/me")
    public UserContext.CurrentUser me() {
        return UserContext.get();
    }
}
