package com.gallant.collab.common;

import lombok.AllArgsConstructor;
import lombok.Data;

import java.util.List;

@Data
@AllArgsConstructor
public class PageResult<T> {
    private List<T> items;
    private long total;
    private int page;
    private int size;
}
