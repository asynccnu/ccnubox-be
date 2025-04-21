CREATE TABLE student_course (
                                stu_id VARCHAR(20) NOT NULL,                    -- 学号
                                cla_id VARCHAR(255) NOT NULL,                    -- 课程ID
                                year VARCHAR(5) NOT NULL,                        -- 学年
                                semester VARCHAR(1) NOT NULL,                    -- 学期
                                is_manually_added BOOLEAN DEFAULT FALSE,         -- 是否为手动添加
                                created_at TIMESTAMP,                            -- 创建时间
                                updated_at TIMESTAMP,                            -- 更新时间

    -- 联合唯一索引
                                UNIQUE INDEX idx_sc (stu_id, cla_id, year, semester, is_manually_added) -- 唯一索引，确保每个学生在每个学年学期只选择一次课程
);
