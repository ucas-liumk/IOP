package com.gallant.collab.service;

import com.baomidou.mybatisplus.core.conditions.query.QueryWrapper;
import com.gallant.collab.common.BizException;
import com.gallant.collab.common.UserContext;
import com.gallant.collab.config.MinioConfig;
import com.gallant.collab.domain.Attachment;
import com.gallant.collab.mapper.AttachmentMapper;
import io.minio.GetObjectArgs;
import io.minio.MinioClient;
import io.minio.PutObjectArgs;
import io.minio.RemoveObjectArgs;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.InputStream;
import java.time.LocalDateTime;
import java.util.List;
import java.util.UUID;

@Slf4j
@Service
@RequiredArgsConstructor
public class FileStorageService {

    private final MinioClient client;
    private final MinioConfig config;
    private final AttachmentMapper mapper;

    /** 后端中转上传 */
    public Attachment upload(String problemId, String stage, MultipartFile file) {
        if (file.isEmpty()) throw new BizException("文件为空");
        String objectKey = problemId + "/" + stage + "/" + UUID.randomUUID() + "-" + safeName(file.getOriginalFilename());
        try (InputStream in = file.getInputStream()) {
            client.putObject(PutObjectArgs.builder()
                    .bucket(config.getBucket())
                    .object(objectKey)
                    .stream(in, file.getSize(), -1)
                    .contentType(file.getContentType())
                    .build());
        } catch (Exception e) {
            throw new BizException("上传失败: " + e.getMessage());
        }

        UserContext.CurrentUser u = UserContext.get();
        Attachment a = new Attachment();
        a.setProblemId(problemId);
        a.setStage(stage);
        a.setFileName(file.getOriginalFilename());
        a.setFileSize(file.getSize());
        a.setContentType(file.getContentType());
        a.setObjectKey(objectKey);
        a.setUploaderId(u.id());
        a.setUploaderName(u.name());
        a.setUploadedAt(LocalDateTime.now());
        mapper.insert(a);
        return a;
    }

    public InputStream download(Long id) {
        Attachment a = mapper.selectById(id);
        if (a == null) throw new BizException("附件不存在");
        try {
            return client.getObject(GetObjectArgs.builder()
                    .bucket(config.getBucket())
                    .object(a.getObjectKey())
                    .build());
        } catch (Exception e) {
            throw new BizException("下载失败: " + e.getMessage());
        }
    }

    public Attachment requireMeta(Long id) {
        Attachment a = mapper.selectById(id);
        if (a == null) throw new BizException("附件不存在");
        return a;
    }

    public void delete(Long id) {
        Attachment a = requireMeta(id);
        try {
            client.removeObject(RemoveObjectArgs.builder()
                    .bucket(config.getBucket()).object(a.getObjectKey()).build());
        } catch (Exception e) {
            log.warn("从 MinIO 删除失败 (将继续删除元数据): {}", e.getMessage());
        }
        mapper.deleteById(id);
    }

    public List<Attachment> listByProblem(String problemId) {
        return mapper.selectList(new QueryWrapper<Attachment>()
                .eq("problem_id", problemId).orderByDesc("uploaded_at"));
    }

    private String safeName(String name) {
        if (name == null) return "file";
        return name.replaceAll("[\\\\/:*?\"<>|]", "_");
    }
}
